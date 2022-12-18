package namerouter

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func (n *NameRouter) hostHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" {
			http.Error(w, "missing Host header", http.StatusBadRequest)
			n.logger.Error("host header not configured",
				zap.String("request host", r.Host),
			)
			return
		}
		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

func (n *NameRouter) externalToHTTPSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])

		if !ip.IsPrivate() {
			newURI := "https://" + r.Host + r.URL.String()
			http.Redirect(w, r, newURI, http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
