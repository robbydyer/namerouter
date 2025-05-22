package namerouter

import (
	"net/http"

	"go.uber.org/zap"
)

type nameHostCtxKeyType string

var nameHostCtxKey nameHostCtxKeyType

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
