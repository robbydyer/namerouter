package tinyauth

import (
	"context"
	"errors"
	"net/http"
)

func CheckAuth(authServer string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		resp, err := http.Get(authServer)
		if err != nil {
			return errors.New("failed to determine auth")
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		return errors.New("unauthorized")
	}
}
