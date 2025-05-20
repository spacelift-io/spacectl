package session

import (
	"net/http"
)

// Defaults returns a HTTP client to use by clients that don't need any further configuration.
func Defaults() *http.Client {
	return http.DefaultClient
}
