package namerouter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/time/rate"
)

type NameRouter struct {
	svr              *http.Server
	httpSvr          *http.Server
	healthSvr        *http.Server
	logger           *zap.Logger
	nameHosts        map[string]*Namehost
	defaultRoute     map[string]*httputil.ReverseProxy
	visitors         map[string]*visitor
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	config           *Config
	sync.Mutex
}

type Config struct {
	RateLimits *RateLimits `yaml:"rateLimits"`
	Routes     []*Namehost `yaml:"routes"`
}

type RateLimits struct {
	Internal *RateLimitConfig `yaml:"internal"`
	External *RateLimitConfig `yaml:"external"`
}
type RateLimitConfig struct {
	Rate  rate.Limit `yaml:"rate"`
	Burst int        `yaml:"burst"`
}

type Namehost struct {
	InternalHosts   []string `yaml:"internal"`
	ExternalHosts   []string `yaml:"external"`
	DestinationAddr string   `yaml:"destination"`
	SourcePort      *string  `yaml:"sourcePort"`
	Always404       bool     `yaml:"always404"`
	proxy           *httputil.ReverseProxy
}

func New(config *Config) (*NameRouter, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	zap.RedirectStdLog(logger)

	n := &NameRouter{
		nameHosts:    make(map[string]*Namehost),
		logger:       logger,
		visitors:     make(map[string]*visitor),
		config:       config,
		defaultRoute: make(map[string]*httputil.ReverseProxy),
	}

	n.config.setDefaults()

	n.backgroundCtx, n.backgroundCancel = context.WithCancel(context.Background())

	go n.visitorCleanup(n.backgroundCtx)

	router := mux.NewRouter()

	router.PathPrefix("/").HandlerFunc(n.handler)

	router.Use(
		n.rateLimiter,
		n.hostHeaderMiddleware,
	)

	aCert := &autocert.Manager{
		Cache:      autocert.DirCache("/cert_cache"),
		Prompt:     autocert.AcceptTOS,
		Email:      "robby.dyer@gmail.com",
		HostPolicy: autocert.HostWhitelist(getExternalHosts(config.Routes)...),
	}

	httpRouter := mux.NewRouter()
	httpRouter.PathPrefix("/").HandlerFunc(n.handler)
	httpRouter.Use(
		n.rateLimiter,
		n.sourcePort,
		n.hostHeaderMiddleware,
		n.externalToHTTPSMiddleware,
	)

	n.svr = &http.Server{
		Addr:      ":https",
		Handler:   router,
		TLSConfig: aCert.TLSConfig(),
		ConnState: n.captureClosedConnIP,
	}

	n.httpSvr = &http.Server{
		Addr:    ":http",
		Handler: httpRouter,
	}

	n.healthSvr = &http.Server{
		Addr: ":9000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("OK"))
		}),
	}

	go func() {
		if err := n.httpSvr.ListenAndServe(); err != nil {
			n.logger.Error("http server failed", zap.Error(err))
		}
	}()
	go func() {
		if err := n.healthSvr.ListenAndServe(); err != nil {
			n.logger.Error("http health server failed", zap.Error(err))
		}
	}()

	for _, nh := range config.Routes {
		if err := n.addNamehost(nh); err != nil {
			return nil, err
		}
	}

	return n, nil
}

func (c *Config) setDefaults() {
	if c.RateLimits == nil {
		c.RateLimits = &RateLimits{}
	}

	if c.RateLimits.External == nil {
		c.RateLimits.External = &RateLimitConfig{
			Rate:  10,
			Burst: 10,
		}
	}

	if c.RateLimits.Internal == nil {
		c.RateLimits.Internal = &RateLimitConfig{
			Rate:  1000,
			Burst: 1000,
		}
	}
}

func (n *NameRouter) Start() error {
	return n.svr.ListenAndServeTLS("", "")
}

func (n *NameRouter) Shutdown(ctx context.Context) {
	n.backgroundCancel()
	_ = n.svr.Shutdown(ctx)
	_ = n.httpSvr.Shutdown(ctx)
	_ = n.healthSvr.Shutdown(ctx)
}

func (n *NameRouter) addNamehost(nh *Namehost) error {
	hosts := []string{}
	hosts = append(hosts, nh.ExternalHosts...)
	hosts = append(hosts, nh.InternalHosts...)

	if nh.DestinationAddr == "" {
		nh.DestinationAddr = "devnull"
	}

	for _, host := range hosts {
		if _, ok := n.nameHosts[host]; ok {
			return fmt.Errorf("host already registered")
		}
	}

	if !nh.Always404 {
		u, err := url.Parse(nh.DestinationAddr)
		if err != nil {
			return fmt.Errorf("failed to parse URL for destination host: %w", err)
		}
		nh.proxy = httputil.NewSingleHostReverseProxy(u)
	}

	for _, host := range hosts {
		n.logger.Info("register host",
			zap.String("host", host),
			zap.String("destination", nh.DestinationAddr),
		)
		if host == "default" {
			n.logger.Info("registering default route",
				zap.String("destination", nh.DestinationAddr),
			)
			if nh.SourcePort != nil {
				n.defaultRoute[*nh.SourcePort] = nh.proxy
			} else {
				n.defaultRoute["80"] = nh.proxy
				n.defaultRoute["443"] = nh.proxy
			}
		}
		n.nameHosts[host] = nh
	}

	return nil
}

func (n *NameRouter) handler(w http.ResponseWriter, r *http.Request) {
	nh, ok := n.nameHosts[r.Host]
	if !ok {
		n.logger.Error("missing proxy config",
			zap.String("request host", r.Host),
		)
		dr, ok := n.defaultRoute["80"]
		if ok && dr != nil {
			n.logger.Info("using default route")
			dr.ServeHTTP(w, r)
			return
		}
		http.Error(w, "host not configured "+r.Host, http.StatusBadRequest)
		return
	}

	if nh.Always404 {
		http.Error(w, "go away", http.StatusNotFound)
		return
	}

	if nh.proxy == nil {
		n.logger.Error("proxy not configured for host",
			zap.String("host", r.Host),
		)
		return
	}

	n.logger.Info("forward request",
		zap.String("Host", r.Host),
		zap.String("source", r.RemoteAddr),
		zap.String("Destination Addr", nh.DestinationAddr),
		zap.String("Request", r.RequestURI),
	)
	nh.proxy.ServeHTTP(w, r)
}
