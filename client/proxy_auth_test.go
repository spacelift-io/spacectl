package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client/session"
)

// responseSpec describes a canned response for recordingTransport.
type responseSpec struct {
	status int
	header http.Header
}

// recordingTransport records every request it receives and replies with the
// queued responses in order, falling back to 200 once the queue is exhausted.
type recordingTransport struct {
	reqs   []*http.Request
	bodies []string
	resps  []responseSpec
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.reqs = append(rt.reqs, req)

	body := ""
	if req.Body != nil && req.Body != http.NoBody {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = string(b)
	}
	rt.bodies = append(rt.bodies, body)

	spec := responseSpec{status: http.StatusOK, header: http.Header{}}
	if len(rt.resps) > 0 {
		spec = rt.resps[0]
		rt.resps = rt.resps[1:]
	}
	return &http.Response{StatusCode: spec.status, Header: spec.header, Body: http.NoBody}, nil
}

func (rt *recordingTransport) last() *http.Request {
	return rt.reqs[len(rt.reqs)-1]
}

func newTestRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://spacelift.example.com/graphql", reader)
	require.NoError(t, err)
	return req
}

func iapChallenge() responseSpec {
	return responseSpec{
		status: http.StatusUnauthorized,
		header: http.Header{"Www-Authenticate": []string{`Bearer realm="cloud_iap"`}},
	}
}

// tokenSequence returns a fetch function that hands out the given tokens in
// order and counts how often it was called.
func tokenSequence(tokens ...string) (func(context.Context) (string, error), *int) {
	calls := 0
	return func(context.Context) (string, error) {
		idx := calls
		calls++
		if idx >= len(tokens) {
			idx = len(tokens) - 1
		}
		return tokens[idx], nil
	}, &calls
}

func captureWarnings(t *testing.T) *[]string {
	t.Helper()

	var warnings []string
	orig := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	t.Cleanup(func() { warnProxyAuth = orig })
	return &warnings
}

func TestWrapWithProxyAuth_NilConfig(t *testing.T) {
	inner := &recordingTransport{}
	assert.Same(t, inner, WrapWithProxyAuth(inner, nil))
}

func TestWrapWithProxyAuth_EmptyCommand(t *testing.T) {
	inner := &recordingTransport{}
	assert.Same(t, inner, WrapWithProxyAuth(inner, &session.ProxyAuthConfig{}))
}

func TestWrapWithProxyAuth_ExposesBase(t *testing.T) {
	inner := &recordingTransport{}
	wrapped := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{Command: []string{"/nonexistent/auth"}})

	transport, ok := wrapped.(*ProxyAuthTransport)
	require.True(t, ok, "expected *ProxyAuthTransport, got %T", wrapped)
	assert.Same(t, inner, transport.Base())
}

func TestProxyAuthTransport_SetsHeaderAndPreservesAuthorization(t *testing.T) {
	inner := &recordingTransport{}
	fetch, _ := tokenSequence("proxy-token")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	req := newTestRequest(t, `{"query":"{ viewer { id } }"}`)
	req.Header.Set("Authorization", "Bearer spacelift-jwt")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "Bearer proxy-token", inner.last().Header.Get("Proxy-Authorization"))
	assert.Equal(t, "Bearer spacelift-jwt", inner.last().Header.Get("Authorization"))
	assert.Equal(t, `{"query":"{ viewer { id } }"}`, inner.bodies[0])
}

func TestProxyAuthTransport_DoesNotMutateOriginalRequest(t *testing.T) {
	inner := &recordingTransport{}
	fetch, _ := tokenSequence("proxy-token")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	req := newTestRequest(t, "")
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, req.Header.Get("Proxy-Authorization"))
}

func TestProxyAuthTransport_FetchFailureIsAnError(t *testing.T) {
	inner := &recordingTransport{}
	transport := &ProxyAuthTransport{
		base: inner,
		fetchToken: func(context.Context) (string, error) {
			return "", errors.New("command \"/x\" not found. Install x first.")
		},
	}

	resp, err := transport.RoundTrip(newTestRequest(t, ""))
	closeIfPresent(resp)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "proxy auth: ")
	assert.Contains(t, err.Error(), "Install x first.")
	assert.Empty(t, inner.reqs, "no request should be sent without a token")
}

func TestProxyAuthTransport_FailuresAreNotCached(t *testing.T) {
	inner := &recordingTransport{}
	calls := 0
	transport := &ProxyAuthTransport{
		base: inner,
		fetchToken: func(context.Context) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("transient")
			}
			return "token", nil
		},
	}

	resp, err := transport.RoundTrip(newTestRequest(t, ""))
	closeIfPresent(resp)
	require.Error(t, err)

	resp, err = transport.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 2, calls)
	assert.Equal(t, "Bearer token", inner.last().Header.Get("Proxy-Authorization"))
}

func TestProxyAuthTransport_TokenIsCachedWhileAccepted(t *testing.T) {
	inner := &recordingTransport{}
	fetch, calls := tokenSequence("token")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	for i := 0; i < 10; i++ {
		resp, err := transport.RoundTrip(newTestRequest(t, ""))
		require.NoError(t, err)
		resp.Body.Close()
	}

	assert.Equal(t, 1, *calls)
}

func TestProxyAuthTransport_RejectionRefreshesTokenAndRetries(t *testing.T) {
	warnings := captureWarnings(t)
	inner := &recordingTransport{resps: []responseSpec{iapChallenge()}}
	fetch, calls := tokenSequence("stale", "fresh")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	req := newTestRequest(t, `{"query":"q"}`)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, *calls)
	require.Len(t, inner.reqs, 2)
	assert.Equal(t, "Bearer stale", inner.reqs[0].Header.Get("Proxy-Authorization"))
	assert.Equal(t, "Bearer fresh", inner.reqs[1].Header.Get("Proxy-Authorization"))
	assert.Equal(t, `{"query":"q"}`, inner.bodies[1], "retried request must carry the body again")
	assert.Empty(t, *warnings)

	// The fresh token is now the cached one.
	resp, err = transport.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "Bearer fresh", inner.last().Header.Get("Proxy-Authorization"))
	assert.Equal(t, 2, *calls)
}

func TestProxyAuthTransport_SameTokenAfterRejectionWarnsOnce(t *testing.T) {
	warnings := captureWarnings(t)
	inner := &recordingTransport{resps: []responseSpec{iapChallenge(), iapChallenge(), iapChallenge()}}
	fetch, _ := tokenSequence("stale")
	transport := &ProxyAuthTransport{base: inner, command: "/usr/local/bin/auth", fetchToken: fetch}

	for i := 0; i < 3; i++ {
		resp, err := transport.RoundTrip(newTestRequest(t, "body"))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "original rejection is returned to the caller")
	}

	require.Len(t, *warnings, 1)
	assert.Contains(t, (*warnings)[0], `"/usr/local/bin/auth"`)
}

func TestProxyAuthTransport_RetriedRequestStillRejectedWarns(t *testing.T) {
	warnings := captureWarnings(t)
	inner := &recordingTransport{resps: []responseSpec{iapChallenge(), iapChallenge()}}
	fetch, _ := tokenSequence("stale", "also-bad")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	resp, err := transport.RoundTrip(newTestRequest(t, "body"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Len(t, inner.reqs, 2)
	assert.Len(t, *warnings, 1)
}

func TestProxyAuthTransport_NonReplayableBodyIsNotRetried(t *testing.T) {
	warnings := captureWarnings(t)
	inner := &recordingTransport{resps: []responseSpec{iapChallenge()}}
	fetch, calls := tokenSequence("stale", "fresh")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	req := newTestRequest(t, "")
	req.Body = io.NopCloser(strings.NewReader("streamed"))
	req.GetBody = nil

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Len(t, inner.reqs, 1)
	assert.Equal(t, 1, *calls)
	assert.Len(t, *warnings, 1)
}

func TestProxyAuthTransport_NoWarnOnSuccess(t *testing.T) {
	warnings := captureWarnings(t)
	inner := &recordingTransport{}
	fetch, _ := tokenSequence("token")
	transport := &ProxyAuthTransport{base: inner, fetchToken: fetch}

	resp, err := transport.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, *warnings)
}

func TestProxyAuthTransport_ConcurrentRequestsFetchOnce(t *testing.T) {
	inner := &recordingTransport{}
	var mu sync.Mutex
	calls := 0
	transport := &ProxyAuthTransport{
		base: &lockedTransport{inner: inner, mu: &mu},
		fetchToken: func(context.Context) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			time.Sleep(5 * time.Millisecond)
			return "token", nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := transport.RoundTrip(newTestRequest(t, ""))
			if assert.NoError(t, err) {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, calls)
}

// lockedTransport serialises access to a recordingTransport for concurrent tests.
type lockedTransport struct {
	inner *recordingTransport
	mu    *sync.Mutex
}

func (l *lockedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inner.RoundTrip(req)
}

func TestLooksLikeProxyAuthRejection(t *testing.T) {
	req := newTestRequest(t, "")

	cases := []struct {
		name string
		resp *http.Response
		want bool
	}{
		{"nil response", nil, false},
		{"200", &http.Response{StatusCode: 200, Header: http.Header{}}, false},
		{"407", &http.Response{StatusCode: http.StatusProxyAuthRequired, Header: http.Header{}}, true},
		{"401 with challenge", &http.Response{
			StatusCode: 401,
			Header:     http.Header{"Www-Authenticate": []string{`Bearer realm="cloud_iap"`}},
		}, true},
		{"401 without challenge", &http.Response{StatusCode: 401, Header: http.Header{}}, true},
		{"403", &http.Response{StatusCode: 403, Header: http.Header{}}, false},
		{"302 to another host", &http.Response{
			StatusCode: 302,
			Header:     http.Header{"Location": []string{"https://accounts.google.com/o/oauth2/v2/auth?x=y"}},
		}, true},
		{"302 to same host", &http.Response{
			StatusCode: 302,
			Header:     http.Header{"Location": []string{"https://spacelift.example.com/login"}},
		}, false},
		{"302 relative", &http.Response{
			StatusCode: 302,
			Header:     http.Header{"Location": []string{"/login"}},
		}, false},
		{"302 same host different case", &http.Response{
			StatusCode: 302,
			Header:     http.Header{"Location": []string{"https://SPACELIFT.example.com/"}},
		}, false},
		{"302 without location", &http.Response{StatusCode: 302, Header: http.Header{}}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, looksLikeProxyAuthRejection(req, tc.resp))
		})
	}
}

func TestFetchTokenFromCommand_NotFoundIncludesSetupHint(t *testing.T) {
	_, err := fetchTokenFromCommand(context.Background(), []string{"/nonexistent/test-auth", "--cached"}, "Install test-auth first.")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"/nonexistent/test-auth" not found`)
	assert.Contains(t, err.Error(), "Install test-auth first.")
}

func TestFetchTokenFromCommand_TrimsOutput(t *testing.T) {
	sh := requireShell(t)

	token, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "printf '  my-token \\n'"}, "")
	require.NoError(t, err)
	assert.Equal(t, "my-token", token)
}

func TestFetchTokenFromCommand_EmptyOutputIsAnError(t *testing.T) {
	sh := requireShell(t)

	_, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "true"}, "Run setup.")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned no token")
	assert.Contains(t, err.Error(), "Run setup.")
}

func TestFetchTokenFromCommand_NonZeroExitIsAnError(t *testing.T) {
	sh := requireShell(t)

	_, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "exit 3"}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestFetchTokenFromCommand_HonoursContextTimeout(t *testing.T) {
	sh := requireShell(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := fetchTokenFromCommand(ctx, []string{sh, "-c", "sleep 5"}, "")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 3*time.Second)
}

func closeIfPresent(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

func requireShell(t *testing.T) string {
	t.Helper()

	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	return sh
}
