package headers

import (
	"context"
	"net/http"
)

type headersKey struct{}

type headerInjectingTransport struct {
	wraps http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (h *headerInjectingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if value := r.Context().Value(headersKey{}); value != nil {
		headers := value.(map[string]string)
		for k, v := range headers {
			r.Header.Set(k, v)
		}
	}
	return h.wraps.RoundTrip(r)
}

// WrapClient wraps an HTTP client to fill in http headers based on the context.
func WrapClient(client *http.Client) *http.Client {
	newClient := *client
	wrappedTransport := newClient.Transport
	if wrappedTransport == nil {
		wrappedTransport = http.DefaultTransport
	}
	newClient.Transport = &headerInjectingTransport{wraps: wrappedTransport}

	return &newClient
}

// WithHTTPHeaders add http headers to the context which will be read by the header injecting http client.
func WithHTTPHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, headersKey{}, headers)
}
