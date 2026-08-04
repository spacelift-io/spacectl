package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client/session"
)

type fakeSession struct {
	endpoint string
}

func (f *fakeSession) BearerToken(context.Context) (string, error) {
	return "test-token", nil
}

func (f *fakeSession) Endpoint() string {
	return f.endpoint
}

func (f *fakeSession) Type() session.CredentialsType {
	return session.CredentialsTypeAPIToken
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestClientDo_EndpointTrimming(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		expectScheme string
		expectHost   string
	}{
		{
			name:         "endpoint with /graphql suffix",
			endpoint:     "https://unit.app.spacelift.io/graphql",
			expectScheme: "https",
			expectHost:   "unit.app.spacelift.io",
		},
		{
			// Regression test: strings.TrimRight previously trimmed any
			// trailing characters found in the "/graphql" cutset, which
			// mangled hosts that don't end in the literal "/graphql"
			// suffix but happen to end in letters from that set (e.g.
			// the trailing "rg" in "org").
			name:         "endpoint without /graphql suffix is left untouched",
			endpoint:     "https://spacelift.example.org",
			expectScheme: "https",
			expectHost:   "spacelift.example.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured *http.Request

			wraps := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					captured = req
					return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
				}),
			}

			c := New(wraps, &fakeSession{endpoint: tt.endpoint})

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://placeholder.invalid/some/path", nil)
			require.NoError(t, err)

			resp, err := c.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.NotNil(t, captured)
			assert.Equal(t, tt.expectScheme, captured.URL.Scheme)
			assert.Equal(t, tt.expectHost, captured.URL.Host)
			assert.Equal(t, "/some/path", captured.URL.Path)
		})
	}
}
