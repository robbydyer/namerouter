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
		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

func (n *NameRouter) rateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			n.logger.Error("failed to parse remote addr",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		limiter := n.getVisitor(ip)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

func (n *NameRouter) captureClosedConnIP(conn net.Conn, state http.ConnState) {
	if state == http.StateClosed || state == http.StateHijacked {
		if conn.RemoteAddr() != nil {
			parts := strings.Split(conn.RemoteAddr().String(), ":")
			if len(parts) > 0 {
				ip := net.ParseIP(parts[0])
				if !ip.IsPrivate() {
					n.logger.Info("closed remote connection",
						zap.String("IP", ip.String()),
						zap.String("local addr", conn.LocalAddr().String()),
						zap.String("Conn state", state.String()),
					)
				}
			}
		}
	}
}
