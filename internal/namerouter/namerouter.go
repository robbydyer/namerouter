package namerouter

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type NameRouter struct {
	svr       *http.Server
	logger    *zap.Logger
	nameHosts map[string]*Namehost
}

type Namehost struct {
	Hosts           []string
	DestinationAddr string
	proxy           *httputil.ReverseProxy
}

func New() (*NameRouter, error) {
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

	n.svr = &http.Server{
		Addr:    "0.0.0.0:80",
		Handler: router,
	}

	return n, nil
}

func (n *NameRouter) Start() error {
	return n.svr.ListenAndServe()
}

func (n *NameRouter) AddNamehost(nh *Namehost) error {
	for _, host := range nh.Hosts {
		if _, ok := n.nameHosts[host]; ok {
			return fmt.Errorf("host already registered")
		}
	}

	u, err := url.Parse("http://" + nh.DestinationAddr)
	if err != nil {
		return fmt.Errorf("failed to parse URL for destination host: %w", err)
	}
	nh.proxy = httputil.NewSingleHostReverseProxy(u)

	for _, host := range nh.Hosts {
		n.nameHosts[host] = nh
	}

	return nil
}

func (n *NameRouter) handler(w http.ResponseWriter, r *http.Request) {
	nh, ok := n.nameHosts[r.Host]
	if !ok {
		http.Error(w, "host not configured "+r.Host, http.StatusBadRequest)
		n.logger.Error("missing proxy config",
			zap.String("request host", r.Host),
		)
		return
	}

	n.logger.Info("forward request",
		zap.String("Host", r.Host),
		zap.String("Destination Addr", nh.DestinationAddr),
	)
	nh.proxy.ServeHTTP(w, r)
}

func (n *NameRouter) hostHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" {
			http.Error(w, "missing Host header", http.StatusBadRequest)
			n.logger.Error("host header not configured",
				zap.String("request host", r.Host),
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}
