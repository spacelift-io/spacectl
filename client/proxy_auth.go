package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spacelift-io/spacectl/client/session"
)

// proxyAuthCommandTimeout bounds how long a single run of the proxy auth
// command may take. A helper that blocks (for example one that opens a browser
// and waits for the user) must not hang spacectl indefinitely.
const proxyAuthCommandTimeout = 30 * time.Second

// ProxyAuthTransport injects a Proxy-Authorization Bearer token into outgoing
// requests. The token is sourced from a configurable command and cached for
// as long as the proxy keeps accepting it. When the proxy rejects a request,
// the cached token is discarded, the command is run again, and the request is
// retried once with the fresh token.
type ProxyAuthTransport struct {
	base       http.RoundTripper
	command    string
	fetchToken func(ctx context.Context) (string, error)

	mu    sync.Mutex
	token string

	warnOnce sync.Once
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

func newProxyAuthTransport(base http.RoundTripper, cfg *session.ProxyAuthConfig) *ProxyAuthTransport {
	command := append([]string(nil), cfg.Command...)
	return &ProxyAuthTransport{
		base:    base,
		command: command[0],
		fetchToken: func(ctx context.Context) (string, error) {
			return fetchTokenFromCommand(ctx, command, cfg.SetupHint)
		},
	}
}

// Base returns the transport wrapped by t.
func (t *ProxyAuthTransport) Base() http.RoundTripper {
	return t.base
}

// RoundTrip implements http.RoundTripper.
func (t *ProxyAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.getToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("proxy auth: %w", err)
	}

	resp, err := t.send(req, token, false)
	if err != nil || !looksLikeProxyAuthRejection(req, resp) {
		return resp, err
	}

	// The proxy rejected a token we sent, most likely because it expired
	// between runs. Fetch a fresh one and retry once, provided the request
	// body can be replayed.
	if !isReplayable(req) {
		t.warnRejected()
		return resp, nil
	}

	t.invalidate(token)
	fresh, err := t.getToken(req.Context())
	if err != nil || fresh == token {
		t.warnRejected()
		return resp, nil
	}

	discardBody(resp)

	resp, err = t.send(req, fresh, true)
	if err == nil && looksLikeProxyAuthRejection(req, resp) {
		t.warnRejected()
	}
	return resp, err
}

// send clones req, attaches the token and forwards it to the base transport.
// When resetBody is set the body is re-created via GetBody, which is required
// when the request is being retried.
func (t *ProxyAuthTransport) send(req *http.Request, token string, resetBody bool) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Proxy-Authorization", "Bearer "+token)
	if resetBody && req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("proxy auth: could not replay request body: %w", err)
		}
		req.Body = body
	}
	return t.base.RoundTrip(req)
}

// getToken returns the cached token, running the command to obtain one if
// none is cached. Failures are never cached.
func (t *ProxyAuthTransport) getToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token != "" {
		return t.token, nil
	}

	ctx, cancel := context.WithTimeout(ctx, proxyAuthCommandTimeout)
	defer cancel()

	token, err := t.fetchToken(ctx)
	if err != nil {
		return "", err
	}

	t.token = token
	return token, nil
}

// invalidate drops the cached token, but only if it is still the one that was
// rejected, so that a token refreshed concurrently by another request is kept.
func (t *ProxyAuthTransport) invalidate(rejected string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token == rejected {
		t.token = ""
	}
}

func (t *ProxyAuthTransport) warnRejected() {
	t.warnOnce.Do(func() {
		warnProxyAuth("the request was rejected after sending the proxy auth token produced by %q; if the proxy is refusing it, re-run that command to obtain a fresh token", t.command)
	})
}

func fetchTokenFromCommand(ctx context.Context, command []string, setupHint string) (string, error) {
	binPath := command[0]
	if _, err := os.Stat(binPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", withHint(fmt.Errorf("command %q not found", binPath), setupHint)
		}
		return "", fmt.Errorf("command %q: %w", binPath, err)
	}

	cmd := exec.CommandContext(ctx, binPath, command[1:]...)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", withHint(fmt.Errorf("command %q failed: %w", binPath, err), setupHint)
	}

	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", withHint(fmt.Errorf("command %q returned no token", binPath), setupHint)
	}

	return token, nil
}

func withHint(err error, hint string) error {
	if hint == "" {
		return err
	}
	return fmt.Errorf("%w. %s", err, hint)
}

// looksLikeProxyAuthRejection reports whether resp may come from a reverse
// proxy refusing the request for lack of a valid proxy credential. It is
// deliberately provider-agnostic and errs on the side of a false positive,
// since the only consequence of one is a single token refresh and retry:
//   - 407 Proxy Authentication Required
//   - 401 Unauthorized (proxies such as GCP IAP answer API calls with a bare
//     401 and no WWW-Authenticate challenge, so the header cannot be required)
//   - a redirect to a different host (typically an SSO login page)
func looksLikeProxyAuthRejection(req *http.Request, resp *http.Response) bool {
	if resp == nil {
		return false
	}

	switch {
	case resp.StatusCode == http.StatusProxyAuthRequired, resp.StatusCode == http.StatusUnauthorized:
		return true
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		location, err := req.URL.Parse(resp.Header.Get("Location"))
		if err != nil || location.Host == "" {
			return false
		}
		return !strings.EqualFold(location.Hostname(), req.URL.Hostname())
	}

	return false
}

// isReplayable reports whether req can be sent a second time.
func isReplayable(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

func discardBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
