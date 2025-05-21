package namerouter

import (
	"net/http"

	"go.uber.org/zap"
)

const nameHostCtxKey = "namehost"

func namehostFromCtx(req *http.Request) *Namehost {
	nh, ok := req.Context().Value(nameHostCtxKey).(*Namehost)
	if ok {
		return nh
	}

	return nil
}

func (n *NameRouter) errNamehostCtx(w http.ResponseWriter) {
	n.logger.Error("failed to get namehost from request context",
		zap.String("host", r.Host),
	)
	http.Error(w, "unknown namehost", http.StatusInternalServerError)
}
