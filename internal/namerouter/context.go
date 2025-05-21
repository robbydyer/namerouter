package namerouter

import (
	"context"
	"net/http"
)

const nameHostCtxKey = "namehost"

func namehostFromCtx(req *http.Request) *Namehost {
	ctxAny := req.Context().Value(nameHostCtxKey)
	nh, ok := ctxAny.(*Namehost)
	if ok {
		return nh
	}

	return nil
}

func setNamehostCtx(req *http.Request, nh *Namehost) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), nameHostCtxKey, nh))
}
