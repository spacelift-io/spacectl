package client

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/spacelift-io/spacectl/client/session"
)

// proxyAuthTransport injects a Proxy-Authorization Bearer token into outgoing
// requests. The token is sourced from a configurable command. If the command
// is unavailable or returns an empty token, the request proceeds without the
// header; if the response then looks like a proxy auth rejection, a one-time
// warning is emitted with the configured hint.
type proxyAuthTransport struct {
	base        http.RoundTripper
	fetchToken  func() (token string, diagnostic string)
	refreshHint string
	warnOnce    sync.Once
}

var warnProxyAuth = func(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "spacectl: warning: "+format+"\n", args...)
}

// WrapWithProxyAuth wraps base with a proxy-auth transport configured by cfg.
// If cfg is nil or has no Command, base is returned unchanged.
func WrapWithProxyAuth(base http.RoundTripper, cfg *session.ProxyAuthConfig) http.RoundTripper {
	if cfg == nil || len(cfg.Command) == 0 {
		return base
	}
	return newProxyAuthTransport(base, cfg)
}

// IsWrappedWithProxyAuth reports whether rt is a proxy-auth transport.
// Exposed for tests in other packages that wire the transport.
func IsWrappedWithProxyAuth(rt http.RoundTripper) bool {
	_, ok := rt.(*proxyAuthTransport)
	return ok
}

// UnwrapProxyAuth returns the base transport inside the proxy-auth wrapper,
// or rt itself if it is not wrapped. Exposed for tests that need to inspect
// the underlying transport (e.g. TLS configuration).
func UnwrapProxyAuth(rt http.RoundTripper) http.RoundTripper {
	if t, ok := rt.(*proxyAuthTransport); ok {
		return t.base
	}
	return rt
}

func newProxyAuthTransport(base http.RoundTripper, cfg *session.ProxyAuthConfig) *proxyAuthTransport {
	setupHint := cfg.SetupHint
	if setupHint == "" {
		setupHint = fmt.Sprintf("proxy auth command %q not found or returned empty output", cfg.Command[0])
	}
	refreshHint := cfg.RefreshHint
	if refreshHint == "" {
		refreshHint = fmt.Sprintf("proxy auth token was rejected; try re-running %q", cfg.Command[0])
	}
	return &proxyAuthTransport{
		base: base,
		fetchToken: memoizeToken(func() (string, string) {
			return fetchTokenFromCommand(cfg.Command, setupHint)
		}),
		refreshHint: refreshHint,
	}
}

func memoizeToken(fetch func() (string, string)) func() (string, string) {
	var (
		once              sync.Once
		token, diagnostic string
	)
	return func() (string, string) {
		once.Do(func() {
			token, diagnostic = fetch()
		})
		return token, diagnostic
	}
}

func fetchTokenFromCommand(command []string, setupHint string) (string, string) {
	binPath := command[0]
	if _, err := os.Stat(binPath); err != nil {
		if os.IsNotExist(err) {
			return "", setupHint
		}
		return "", ""
	}
	var args []string
	if len(command) > 1 {
		args = command[1:]
	}
	out, err := exec.Command(binPath, args...).Output()
	if err != nil {
		return "", fmt.Sprintf(
			"%s failed: %v. %s",
			binPath, err, setupHint,
		)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", setupHint
	}
	return token, ""
}

// looksLikeProxyAuthRejection returns true when resp appears to be from a
// reverse proxy rejecting the request because no valid token was supplied.
// Checks for:
//   - 3xx with Location pointing into accounts.google.com (GCP IAP SSO redirect)
//   - 401 with WWW-Authenticate containing "cloud_iap" (GCP IAP challenge)
//   - 401 with body containing "Invalid IAP credentials" (GCP IAP rejection)
//
// This is intentionally broad enough to catch common reverse proxy patterns.
func looksLikeProxyAuthRejection(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if loc := resp.Header.Get("Location"); strings.Contains(loc, "accounts.google.com") {
			return true
		}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		if wwwAuth := resp.Header.Get("Www-Authenticate"); strings.Contains(wwwAuth, "cloud_iap") {
			return true
		}
	}
	return false
}

func (t *proxyAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var diagnostic string
	var sentToken bool
	token, d := t.fetchToken()
	diagnostic = d
	if token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Proxy-Authorization", "Bearer "+token)
		sentToken = true
	}
	resp, err := t.base.RoundTrip(req)
	if err == nil && looksLikeProxyAuthRejection(resp) {
		t.warnOnce.Do(func() {
			switch {
			case sentToken:
				warnProxyAuth("%s", t.refreshHint)
			case diagnostic != "":
				warnProxyAuth("%s", diagnostic)
			}
		})
	}
	return resp, err
}
