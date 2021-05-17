package headers

import (
	"context"
	"net/http"
)

type headersKey struct{}

type HeaderInjectingTransport struct {
	wraps http.RoundTripper
}

func (h *HeaderInjectingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if value := r.Context().Value(headersKey{}); value != nil {
		headers := value.(map[string]string)
		for k, v := range headers {
			r.Header.Set(k, v)
		}
	}
	return h.wraps.RoundTrip(r)
}

func WrapClient(client *http.Client) *http.Client {
	newClient := *client
	wrappedTransport := newClient.Transport
	if wrappedTransport == nil {
		wrappedTransport = http.DefaultTransport
	}
	newClient.Transport = &HeaderInjectingTransport{wraps: wrappedTransport}

	return &newClient
}

func WithHTTPHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, headersKey{}, headers)
}
