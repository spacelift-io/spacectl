package browserauth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client/session"
)

func TestAuthCallbackIgnoresStrayRequests(t *testing.T) {
	handler, err := BeginWithBindAddress(context.Background(), &session.StoredCredentials{
		Type:     session.CredentialsTypeAPIToken,
		Endpoint: "https://example.app.spacelift.io",
	}, "127.0.0.1", 0)
	require.NoError(t, err)
	t.Cleanup(handler.Cancel)

	base := fmt.Sprintf("http://%s:%d", handler.Host, handler.Port)
	noRedirects := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// Probes a local security agent or browser might send before the real callback.
	for _, probe := range []struct{ method, path string }{
		{http.MethodHead, "/"},
		{http.MethodGet, "/"},
		{http.MethodGet, "/favicon.ico"},
		{http.MethodGet, "/?port=1234"},
	} {
		req, err := http.NewRequestWithContext(t.Context(), probe.method, base+probe.path, nil)
		require.NoError(t, err)

		resp, err := noRedirects.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "%s %s", probe.method, probe.path)
	}

	// None of the probes should have completed the login.
	waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, handler.Wait(waitCtx), context.DeadlineExceeded)
}

func TestAuthCallbackFailsOnMalformedCallback(t *testing.T) {
	handler, err := BeginWithBindAddress(context.Background(), &session.StoredCredentials{
		Type:     session.CredentialsTypeAPIToken,
		Endpoint: "https://example.app.spacelift.io",
	}, "127.0.0.1", 0)
	require.NoError(t, err)
	t.Cleanup(handler.Cancel)

	noRedirects := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// A callback with a token but no key is still a real (broken) callback and should fail.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://%s:%d/?token=abc", handler.Host, handler.Port), nil)
	require.NoError(t, err)

	resp, err := noRedirects.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://example.app.spacelift.io/auth_failure", resp.Header.Get("Location"))

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.EqualError(t, handler.Wait(waitCtx), "missing key parameter")
}
