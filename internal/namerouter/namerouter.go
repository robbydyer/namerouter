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
)

const defaultNameHost = "defaultNamehost"

type NameRouter struct {
	svr              *http.Server
	httpSvr          *http.Server
	healthSvr        *http.Server
	logger           *zap.Logger
	nameHosts        map[string]*Namehost
	defaultRoute     *httputil.ReverseProxy
	visitors         map[string]*visitor
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	sync.Mutex
}

type Config struct {
	Routes []*Namehost `yaml:"routes"`
}

type Namehost struct {
	InternalHosts   []string `yaml:"internal"`
	ExternalHosts   []string `yaml:"external"`
	DestinationAddr string   `yaml:"destination"`
	proxy           *httputil.ReverseProxy
}

func New(config *Config) (*NameRouter, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	zap.RedirectStdLog(logger)

	n := &NameRouter{
		nameHosts: make(map[string]*Namehost),
		logger:    logger,
		visitors:  make(map[string]*visitor),
	}

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

func (n *NameRouter) Start() error {
	return n.svr.ListenAndServeTLS("", "")
}

func (n *NameRouter) Shutdown(ctx context.Context) {
	n.backgroundCancel()
	n.svr.Shutdown(ctx)
	n.httpSvr.Shutdown(ctx)
	n.healthSvr.Shutdown(ctx)
}

func (n *NameRouter) addNamehost(nh *Namehost) error {
	hosts := []string{}
	hosts = append(hosts, nh.ExternalHosts...)
	hosts = append(hosts, nh.InternalHosts...)

	for _, host := range hosts {
		if _, ok := n.nameHosts[host]; ok {
			return fmt.Errorf("host already registered")
		}
	}

	u, err := url.Parse(nh.DestinationAddr)
	if err != nil {
		return fmt.Errorf("failed to parse URL for destination host: %w", err)
	}
	nh.proxy = httputil.NewSingleHostReverseProxy(u)

	for _, host := range hosts {
		n.logger.Info("register host",
			zap.String("host", host),
			zap.String("destination", nh.DestinationAddr),
		)
		if host == "default" {
			n.logger.Info("registering default route",
				zap.String("destination", nh.DestinationAddr),
			)
			n.defaultRoute = nh.proxy
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
		if n.defaultRoute != nil {
			n.logger.Info("using default route")
			n.defaultRoute.ServeHTTP(w, r)
			return
		}
		http.Error(w, "host not configured "+r.Host, http.StatusBadRequest)
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
