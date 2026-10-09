package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/spacelift-io/spacectl/client/session"
)

type recordingTransport struct {
	req  *http.Request
	resp *http.Response
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.req = req
	if rt.resp != nil {
		return rt.resp, nil
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody}, nil
}

func newTestRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://spacelift.example.com/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func testConfig() *session.ProxyAuthConfig {
	return &session.ProxyAuthConfig{
		Command:     []string{"/nonexistent/test-auth", "--cache-only"},
		SetupHint:   "Install test-auth to authenticate.",
		RefreshHint: "Refresh with 'test-auth --refresh'.",
	}
}

func TestWrapWithProxyAuth_NilConfig(t *testing.T) {
	inner := &recordingTransport{}
	result := WrapWithProxyAuth(inner, nil)
	if result != inner {
		t.Error("expected base transport returned unchanged for nil config")
	}
}

func TestWrapWithProxyAuth_EmptyCommand(t *testing.T) {
	inner := &recordingTransport{}
	result := WrapWithProxyAuth(inner, &session.ProxyAuthConfig{})
	if result != inner {
		t.Error("expected base transport returned unchanged for empty command")
	}
}

func TestProxyAuthTransport_WithToken_HeaderSet(t *testing.T) {
	inner := &recordingTransport{}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "test-proxy-token", "" },
	}

	req := newTestRequest(t)
	req.Header.Set("Authorization", "Bearer spacelift-jwt")
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := inner.req.Header.Get("Proxy-Authorization"); got != "Bearer test-proxy-token" {
		t.Errorf("expected Proxy-Authorization 'Bearer test-proxy-token', got %q", got)
	}
	if got := inner.req.Header.Get("Authorization"); got != "Bearer spacelift-jwt" {
		t.Errorf("expected original Authorization header preserved, got %q", got)
	}
}

func TestProxyAuthTransport_NoToken_NoHeader(t *testing.T) {
	inner := &recordingTransport{}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "", "auth tool missing" },
	}

	req := newTestRequest(t)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := inner.req.Header.Get("Proxy-Authorization"); got != "" {
		t.Errorf("expected no Proxy-Authorization when token is empty, got %q", got)
	}
}

func TestProxyAuthTransport_DoesNotMutateOriginalRequest(t *testing.T) {
	inner := &recordingTransport{}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "test-token", "" },
	}

	req := newTestRequest(t)
	origHeaders := req.Header.Clone()
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if req.Header.Get("Proxy-Authorization") != origHeaders.Get("Proxy-Authorization") {
		t.Error("original request was mutated")
	}
}

func TestProxyAuthTransport_WarnsOnRejection(t *testing.T) {
	var warnings []string
	origWarn := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnings = append(warnings, format)
	}
	defer func() { warnProxyAuth = origWarn }()

	inner := &recordingTransport{
		resp: &http.Response{
			StatusCode: 401,
			Header:     http.Header{"Www-Authenticate": []string{`Bearer realm="cloud_iap"`}},
		},
	}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "", "auth tool has no cached token." },
	}

	req := newTestRequest(t)
	_, _ = transport.RoundTrip(req)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestProxyAuthTransport_WarnsOnceOnly(t *testing.T) {
	var warnCount int
	origWarn := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnCount++
	}
	defer func() { warnProxyAuth = origWarn }()

	inner := &recordingTransport{
		resp: &http.Response{
			StatusCode: 401,
			Header:     http.Header{"Www-Authenticate": []string{`Bearer realm="cloud_iap"`}},
		},
	}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "", "missing" },
	}

	for i := 0; i < 5; i++ {
		req := newTestRequest(t)
		_, _ = transport.RoundTrip(req)
	}

	if warnCount != 1 {
		t.Errorf("expected exactly 1 warning across 5 requests, got %d", warnCount)
	}
}

func TestProxyAuthTransport_ExpiredToken_UsesRefreshHint(t *testing.T) {
	var warnings []string
	origWarn := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	defer func() { warnProxyAuth = origWarn }()

	inner := &recordingTransport{
		resp: &http.Response{
			StatusCode: 401,
			Header:     http.Header{"Www-Authenticate": []string{`Bearer realm="cloud_iap"`}},
		},
	}
	transport := &proxyAuthTransport{
		base:        inner,
		fetchToken:  func() (string, string) { return "expired-token", "" },
		refreshHint: "Refresh with 'test-auth --refresh'.",
	}

	req := newTestRequest(t)
	_, _ = transport.RoundTrip(req)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "Refresh") {
		t.Errorf("expected refresh hint in warning, got %q", warnings[0])
	}
}

func TestProxyAuthTransport_NoWarnOn200(t *testing.T) {
	var warnCount int
	origWarn := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnCount++
	}
	defer func() { warnProxyAuth = origWarn }()

	inner := &recordingTransport{}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "", "auth tool missing" },
	}

	req := newTestRequest(t)
	_, _ = transport.RoundTrip(req)

	if warnCount != 0 {
		t.Errorf("expected no warning on 200 response, got %d", warnCount)
	}
}

func TestProxyAuthTransport_RedirectToGoogleSSO(t *testing.T) {
	var warnings []string
	origWarn := warnProxyAuth
	warnProxyAuth = func(format string, args ...interface{}) {
		warnings = append(warnings, format)
	}
	defer func() { warnProxyAuth = origWarn }()

	inner := &recordingTransport{
		resp: &http.Response{
			StatusCode: 302,
			Header:     http.Header{"Location": []string{"https://accounts.google.com/o/oauth2/v2/auth?..."}},
		},
	}
	transport := &proxyAuthTransport{
		base:       inner,
		fetchToken: func() (string, string) { return "", "no token" },
	}

	req := newTestRequest(t)
	_, _ = transport.RoundTrip(req)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning on 302-to-Google-SSO, got %d", len(warnings))
	}
}

func TestIsWrappedWithProxyAuth(t *testing.T) {
	inner := &recordingTransport{}
	wrapped := WrapWithProxyAuth(inner, testConfig())
	if !IsWrappedWithProxyAuth(wrapped) {
		t.Error("expected IsWrappedWithProxyAuth to return true")
	}
	if IsWrappedWithProxyAuth(inner) {
		t.Error("expected IsWrappedWithProxyAuth to return false for unwrapped transport")
	}
}

func TestUnwrapProxyAuth(t *testing.T) {
	inner := &recordingTransport{}
	wrapped := WrapWithProxyAuth(inner, testConfig())
	unwrapped := UnwrapProxyAuth(wrapped)
	if unwrapped != inner {
		t.Error("expected UnwrapProxyAuth to return the inner transport")
	}
	if UnwrapProxyAuth(inner) != inner {
		t.Error("expected UnwrapProxyAuth to return the same transport when not wrapped")
	}
}

func TestMemoizeToken(t *testing.T) {
	callCount := 0
	fetch := memoizeToken(func() (string, string) {
		callCount++
		return "token", ""
	})

	for i := 0; i < 10; i++ {
		tok, _ := fetch()
		if tok != "token" {
			t.Fatalf("expected 'token', got %q", tok)
		}
	}
	if callCount != 1 {
		t.Errorf("expected fetch called once, called %d times", callCount)
	}
}

