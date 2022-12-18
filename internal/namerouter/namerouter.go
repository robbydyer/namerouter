package namerouter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

const defaultNameHost = "defaultNamehost"

type NameRouter struct {
	svr          *http.Server
	httpSvr      *http.Server
	logger       *zap.Logger
	nameHosts    map[string]*Namehost
	defaultRoute *httputil.ReverseProxy
}

type Namehost struct {
	Hosts           []string
	DestinationAddr string
	proxy           *httputil.ReverseProxy
}

func New(nameHosts ...*Namehost) (*NameRouter, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	n := &NameRouter{
		nameHosts: make(map[string]*Namehost),
		logger:    logger,
	}

	router := mux.NewRouter()

	router.PathPrefix("/").HandlerFunc(n.handler)

	router.Use(n.hostHeaderMiddleware)

	aCert := &autocert.Manager{
		Cache:      autocert.DirCache("/cert_cache"),
		Prompt:     autocert.AcceptTOS,
		Email:      "robby.dyer@gmail.com",
		HostPolicy: autocert.HostWhitelist(getHosts(nameHosts)...),
	}

	n.svr = &http.Server{
		Addr:      ":https",
		Handler:   router,
		TLSConfig: aCert.TLSConfig(),
	}

	n.httpSvr = &http.Server{
		Addr: ":http",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// redirect to 443
			newURI := "https://" + r.Host + r.URL.String()
			http.Redirect(w, r, newURI, http.StatusFound)
		}),
	}

	go func() {
		if err := n.httpSvr.ListenAndServe(); err != nil {
			n.logger.Error("http server failed", zap.Error(err))
		}
	}()

	for _, nh := range nameHosts {
		if err := n.addNamehost(nh); err != nil {
			return nil, err
		}
	}

	return n, nil
}

func (n *NameRouter) Start() error {
	return n.svr.ListenAndServe()
}

func (n *NameRouter) Shutdown(ctx context.Context) {
	n.svr.Shutdown(ctx)
	n.httpSvr.Shutdown(ctx)
}

func (n *NameRouter) addNamehost(nh *Namehost) error {
	for _, host := range nh.Hosts {
		if _, ok := n.nameHosts[host]; ok {
			return fmt.Errorf("host already registered")
		}
	}

	u, err := url.Parse(nh.DestinationAddr)
	if err != nil {
		return fmt.Errorf("failed to parse URL for destination host: %w", err)
	}
	nh.proxy = httputil.NewSingleHostReverseProxy(u)

	for _, host := range nh.Hosts {
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
		zap.String("Destination Addr", nh.DestinationAddr),
	)
	nh.proxy.ServeHTTP(w, r)
}
