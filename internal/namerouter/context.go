package namerouter

import (
	"context"
	"net/http"
)

func namehostFromCtx(req *http.Request) *Namehost {
	ctxAny := req.Context().Value("namehost")
	nh, ok := ctxAny.(*Namehost)
	if ok {
		return nh
	}

	return nil
}

func setNamehostCtx(req *http.Request, nh *Namehost) *http.Request {
	ctx := req.Context()
	return req.Clone(context.WithValue(ctx, "namehost", nh))
}
