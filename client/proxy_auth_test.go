package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// testEndpoint is the Spacelift endpoint the test requests are addressed to.
const testEndpoint = "https://spacelift.example.com"

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
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	inner := &recordingTransport{}

	wrapped, err := WrapWithProxyAuth(inner, nil, "", testEndpoint)
	require.NoError(t, err)
	assert.Same(t, inner, wrapped)
}

func TestWrapWithProxyAuth_EmptyConfig(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	inner := &recordingTransport{}

	wrapped, err := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{}, "", testEndpoint)
	require.NoError(t, err)
	assert.Same(t, inner, wrapped)
}

func TestWrapWithProxyAuth_ExposesBase(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	inner := &recordingTransport{}

	wrapped, err := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{Command: []string{"/nonexistent/auth"}}, "", testEndpoint)
	require.NoError(t, err)

	transport, ok := wrapped.(*ProxyAuthTransport)
	require.True(t, ok, "expected *ProxyAuthTransport, got %T", wrapped)
	assert.Same(t, inner, transport.Base())
}

func TestWrapWithProxyAuth_EnvironmentWinsOverConfig(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, " env-token \n")
	inner := &recordingTransport{}

	// The command is relative and would be rejected if it were consulted.
	wrapped, err := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{Command: []string{"auth"}}, "", testEndpoint)
	require.NoError(t, err)

	resp, err := wrapped.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	closeIfPresent(resp)
	assert.Equal(t, "Bearer env-token", inner.last().Header.Get("Proxy-Authorization"))
}

func TestWrapWithProxyAuth_EnvironmentWithoutConfig(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "env-token")
	inner := &recordingTransport{}

	wrapped, err := WrapWithProxyAuth(inner, nil, "", testEndpoint)
	require.NoError(t, err)

	resp, err := wrapped.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	closeIfPresent(resp)
	assert.Equal(t, "Bearer env-token", inner.last().Header.Get("Proxy-Authorization"))
}

func TestWrapWithProxyAuth_TokenFileWinsOverCommand(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	tokenFile := writeTokenFile(t, "file-token\n")
	inner := &recordingTransport{}

	wrapped, err := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{TokenFile: tokenFile, Command: []string{"auth"}}, "", testEndpoint)
	require.NoError(t, err)

	resp, err := wrapped.RoundTrip(newTestRequest(t, ""))
	require.NoError(t, err)
	closeIfPresent(resp)
	assert.Equal(t, "Bearer file-token", inner.last().Header.Get("Proxy-Authorization"))
}

func TestWrapWithProxyAuth_TokenFileIsReReadOnRejection(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	tokenFile := writeTokenFile(t, "stale")
	inner := &recordingTransport{resps: []responseSpec{iapChallenge()}}

	wrapped, err := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{TokenFile: tokenFile}, "", testEndpoint)
	require.NoError(t, err)

	// Simulate an external refresher rewriting the file between the first
	// attempt and the retry: the file transport reads it on every fetch.
	inner.resps = []responseSpec{iapChallenge()}
	rt := wrapped.(*ProxyAuthTransport)
	original := rt.fetchToken
	rt.fetchToken = func(ctx context.Context) (string, error) {
		token, err := original(ctx)
		require.NoError(t, os.WriteFile(tokenFile, []byte("fresh"), 0o600))
		return token, err
	}

	resp, err := wrapped.RoundTrip(newTestRequest(t, "body"))
	require.NoError(t, err)
	closeIfPresent(resp)

	require.Len(t, inner.reqs, 2)
	assert.Equal(t, "Bearer stale", inner.reqs[0].Header.Get("Proxy-Authorization"))
	assert.Equal(t, "Bearer fresh", inner.reqs[1].Header.Get("Proxy-Authorization"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWrapWithProxyAuth_RelativeCommandIsRejected(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")

	_, err := WrapWithProxyAuth(&recordingTransport{}, &session.ProxyAuthConfig{Command: []string{"iap-auth"}}, "", testEndpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an absolute path")
}

func TestWrapWithProxyAuth_WritableConfigFileIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")

	configFile := filepath.Join(t.TempDir(), session.ConfigFileName)
	require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0o600))
	require.NoError(t, os.Chmod(configFile, 0o666))

	_, err := WrapWithProxyAuth(&recordingTransport{}, &session.ProxyAuthConfig{Command: []string{"/usr/local/bin/auth"}}, configFile, testEndpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writable by other users")
}

func TestWrapWithProxyAuth_SymlinkedConfigFileIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")

	dir := t.TempDir()
	target := filepath.Join(dir, "real-config.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o600))
	link := filepath.Join(dir, session.ConfigFileName)
	require.NoError(t, os.Symlink(target, link))

	_, err := WrapWithProxyAuth(&recordingTransport{}, &session.ProxyAuthConfig{Command: []string{"/usr/local/bin/auth"}}, link, testEndpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a symbolic link")
}

func TestWrapWithProxyAuth_InvalidEndpointIsRejected(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "env-token")

	_, err := WrapWithProxyAuth(&recordingTransport{}, nil, "", "not a url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid URL")
}

func TestWrapWithProxyAuth_PrivateConfigFileIsAccepted(t *testing.T) {
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")

	configFile := filepath.Join(t.TempDir(), session.ConfigFileName)
	require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0o600))

	wrapped, err := WrapWithProxyAuth(&recordingTransport{}, &session.ProxyAuthConfig{Command: []string{"/usr/local/bin/auth"}}, configFile, testEndpoint)
	require.NoError(t, err)
	assert.IsType(t, &ProxyAuthTransport{}, wrapped)
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

func TestProxyAuthTransport_OnlySendsTokenToTheEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		url      string
		want     bool
	}{
		{"same host", "https://spacelift.example.com", "https://spacelift.example.com/graphql", true},
		{"same host, different case", "https://spacelift.example.com", "https://SPACELIFT.example.com/graphql", true},
		{"explicit default port", "https://spacelift.example.com", "https://spacelift.example.com:443/graphql", true},
		{"other host", "https://spacelift.example.com", "https://accounts.google.com/o/oauth2/v2/auth", false},
		{"other port", "https://spacelift.example.com", "https://spacelift.example.com:8443/graphql", false},
		{"other scheme", "https://spacelift.example.com", "http://spacelift.example.com/graphql", false},
		{"no endpoint configured", "", "https://anything.example.net/", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, err := parseEndpoint(tc.endpoint)
			require.NoError(t, err)

			inner := &recordingTransport{}
			fetch, calls := tokenSequence("tok")
			transport := newProxyAuthTransport(inner, endpoint, "test", fetch)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tc.url, nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			closeIfPresent(resp)
			require.NoError(t, err)
			require.Len(t, inner.reqs, 1)

			if tc.want {
				assert.Equal(t, "Bearer tok", inner.last().Header.Get("Proxy-Authorization"))
				assert.Equal(t, 1, *calls)
			} else {
				assert.Empty(t, inner.last().Header.Get("Proxy-Authorization"), "the token must not reach another host")
				assert.Equal(t, 0, *calls, "no token should be fetched for another host")
			}
		})
	}
}

// TestProxyAuthTransport_FollowedRedirectDoesNotLeakToken covers the case the
// endpoint host check exists for: the API (or something in front of it) answers
// with a redirect to another host, and http.Client follows it through this
// transport. The other host must not receive the proxy credential.
func TestProxyAuthTransport_FollowedRedirectDoesNotLeakToken(t *testing.T) {
	var elsewhereHeaders []http.Header
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhereHeaders = append(elsewhereHeaders, r.Header.Clone())
		w.WriteHeader(http.StatusOK)
	}))
	defer elsewhere.Close()

	var endpointHeaders []http.Header
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpointHeaders = append(endpointHeaders, r.Header.Clone())
		http.Redirect(w, r, elsewhere.URL+"/login", http.StatusFound)
	}))
	defer endpoint.Close()

	endpointURL, err := url.Parse(endpoint.URL)
	require.NoError(t, err)

	fetch, _ := tokenSequence("tok")
	client := &http.Client{Transport: newProxyAuthTransport(http.DefaultTransport, endpointURL, "test", fetch)}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint.URL+"/graphql", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	closeIfPresent(resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// The endpoint saw the token on the original request and, since the
	// redirect counts as a rejection, on the retry too.
	require.NotEmpty(t, endpointHeaders)
	for _, h := range endpointHeaders {
		assert.Equal(t, "Bearer tok", h.Get("Proxy-Authorization"))
	}

	// The host we were redirected to never did.
	require.Len(t, elsewhereHeaders, 1)
	assert.Empty(t, elsewhereHeaders[0].Get("Proxy-Authorization"), "the proxy credential leaked to the redirect target")
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

	var proxyErr *ProxyAuthError
	assert.True(t, errors.As(fmt.Errorf("wrapped: %w", err), &proxyErr), "callers must be able to recognise a proxy auth failure")
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
	transport := &ProxyAuthTransport{base: inner, source: `command "/usr/local/bin/auth"`, fetchToken: fetch}

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

func TestFetchTokenFromFile_TrimsContents(t *testing.T) {
	token, err := fetchTokenFromFile(writeTokenFile(t, "  file-token \n"))
	require.NoError(t, err)
	assert.Equal(t, "file-token", token)
}

func TestFetchTokenFromFile_NotFound(t *testing.T) {
	_, err := fetchTokenFromFile(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFetchTokenFromFile_EmptyIsAnError(t *testing.T) {
	_, err := fetchTokenFromFile(writeTokenFile(t, "\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestFetchTokenFromCommand_NotFound(t *testing.T) {
	_, err := fetchTokenFromCommand(context.Background(), []string{"/nonexistent/test-auth", "--cached"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"/nonexistent/test-auth" not found`)
}

func TestFetchTokenFromCommand_TrimsOutput(t *testing.T) {
	sh := requireShell(t)

	token, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "printf '  my-token \\n'"})
	require.NoError(t, err)
	assert.Equal(t, "my-token", token)
}

func TestFetchTokenFromCommand_EmptyOutputIsAnError(t *testing.T) {
	sh := requireShell(t)

	_, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "true"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned no token")
}

func TestFetchTokenFromCommand_NonZeroExitIsAnError(t *testing.T) {
	sh := requireShell(t)

	_, err := fetchTokenFromCommand(context.Background(), []string{sh, "-c", "exit 3"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestFetchTokenFromCommand_HonoursContextTimeout(t *testing.T) {
	sh := requireShell(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := fetchTokenFromCommand(ctx, []string{sh, "-c", "sleep 5"})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 3*time.Second)
}

func writeTokenFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "proxy-token")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
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
