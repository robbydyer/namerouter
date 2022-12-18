package namerouter

import (
	"net/http"

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
