package namerouter

import (
	"net/http"

	"go.uber.org/zap"
)

const nameHostCtxKey = "namehost"

func (n *NameRouter) errNamehostCtx(w http.ResponseWriter, r *http.Request) {
	n.logger.Error("failed to get namehost from request context",
		zap.String("host", r.Host),
	)
	http.Error(w, "unknown namehost", http.StatusInternalServerError)
}

func (n *NameRouter) getNamehost(req *http.Request) *Namehost {
	// Check request context first
	nh, ok := req.Context().Value(nameHostCtxKey).(*Namehost)
	if ok {
		n.logger.Debug("got namehost from request context",
			zap.String("host", req.Host),
		)
		return nh
	}

	n.RLock()
	nh, ok = n.nameHosts[req.Host]
	n.RUnlock()
	if ok {
		n.logger.Debug("got namehost from config map",
			zap.String("host", req.Host),
		)
		return nh
	}

	return nil
}
