package session

import (
	"context"
	"net/http"
)

// Defaults returns default context and HTTP client to use by clients that don't
// need any further configuration.
func Defaults() (context.Context, *http.Client) {
	return context.Background(), http.DefaultClient
}
