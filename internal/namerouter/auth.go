package namerouter

import (
	"fmt"
	"net/http"

	"github.com/markbates/goth/gothic"
	"go.uber.org/zap"
)

func (n *NameRouter) authHandler(w http.ResponseWriter, r *http.Request) { // try to get the user without re-authenticating
	if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
		n.logger.Info("got user",
			zap.String("email", gothUser.Email),
		)
		return
	} else {
		gothic.BeginAuthHandler(w, r)
	}
}

func (n *NameRouter) authCallback(w http.ResponseWriter, r *http.Request) {
	var body []byte
	defer r.Body.Close()

	if _, err := r.Body.Read(body); err != nil {
		n.logger.Error("failed to read callback body",
			zap.Error(err),
		)
	}
	n.logger.Info("auth callback",
		zap.ByteString("body", body),
	)

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	n.logger.Info("callback user",
		zap.String("email", user.Email),
	)
}
