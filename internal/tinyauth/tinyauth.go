package tinyauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func CheckAuth(authServer string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		resp, err := http.Get(authServer)
		if err != nil {
			return fmt.Errorf("failed to determine auth: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		return errors.New("unauthorized")
	}
}
