package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fakeGitHubAndSpacelift serves both the GitHub Actions ID token endpoint
// (/token) and the Spacelift GraphQL API (/graphql). Each minted token is
// numbered, so tests can tell a fresh mint from a reused one.
type fakeGitHubAndSpacelift struct {
	*httptest.Server

	mu        sync.Mutex
	minted    int
	audiences []string
	query     []string
	secrets   []string
	tokenCode int
}

func newFakeGitHubAndSpacelift(t *testing.T) *fakeGitHubAndSpacelift {
	t.Helper()

	f := &fakeGitHubAndSpacelift{tokenCode: http.StatusOK}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.URL.Path {
		case "/token":
			if got := r.Header.Get("Authorization"); got != "Bearer request-token" {
				t.Errorf("unexpected Authorization header %q", got)
			}
			f.audiences = append(f.audiences, r.URL.Query().Get("audience"))
			f.query = append(f.query, r.URL.Query().Get("api-version"))

			if f.tokenCode != http.StatusOK {
				w.WriteHeader(f.tokenCode)
				return
			}
			f.minted++
			fmt.Fprintf(w, `{"value":"gh-token-%d"}`, f.minted)

		case "/graphql":
			var body struct {
				Variables struct {
					Secret string `json:"secret"`
				} `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("could not decode GraphQL request: %v", err)
			}
			f.secrets = append(f.secrets, body.Variables.Secret)

			// validUntil in the past keeps the session stale, so every
			// BearerToken call exchanges again.
			fmt.Fprint(w, `{"data":{"apiKeyUser":{"jwt":"SpaceliftJWT","validUntil":123}}}`)

		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
	}))
	t.Cleanup(f.Close)

	return f
}

func (f *fakeGitHubAndSpacelift) env(extra map[string]string) func(string) (string, bool) {
	values := map[string]string{
		EnvSpaceliftAPIKeyEndpoint:          f.URL,
		EnvSpaceliftAPIKeyID:                "key-id",
		envGitHubActionsIDTokenRequestURL:   f.URL + "/token?api-version=2.0",
		envGitHubActionsIDTokenRequestToken: "request-token",
		EnvSpaceliftAPIKeyOIDCProvider:      oidcProviderGitHubActions,
	}
	for k, v := range extra {
		if v == "" {
			delete(values, k)
			continue
		}
		values[k] = v
	}

	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}

func TestFromEnvironmentGitHubActionsOIDC(t *testing.T) {
	t.Run("mints the token with the endpoint host as audience", func(t *testing.T) {
		f := newFakeGitHubAndSpacelift(t)

		sess, err := FromEnvironment(context.Background(), f.Client())(f.env(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.Type() != CredentialsTypeAPIKey {
			t.Errorf("unexpected session type %v", sess.Type())
		}

		if want := []string{"gh-token-1"}; !slices.Equal(f.secrets, want) {
			t.Errorf("exchanged secrets %v, want %v", f.secrets, want)
		}
		if want := []string{"127.0.0.1"}; !slices.Equal(f.audiences, want) {
			t.Errorf("audiences %v, want %v", f.audiences, want)
		}
		if want := []string{"2.0"}; !slices.Equal(f.query, want) {
			t.Errorf("existing query params were not kept: api-version %v, want %v", f.query, want)
		}
	})

	t.Run("uses the audience override", func(t *testing.T) {
		f := newFakeGitHubAndSpacelift(t)

		_, err := FromEnvironment(context.Background(), f.Client())(f.env(map[string]string{
			EnvSpaceliftAPIKeyOIDCAudience: "custom-audience",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []string{"custom-audience"}; !slices.Equal(f.audiences, want) {
			t.Errorf("audiences %v, want %v", f.audiences, want)
		}
	})

	t.Run("mints a new token on every exchange", func(t *testing.T) {
		f := newFakeGitHubAndSpacelift(t)

		sess, err := FromEnvironment(context.Background(), f.Client())(f.env(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := sess.BearerToken(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if want := []string{"gh-token-1", "gh-token-2"}; !slices.Equal(f.secrets, want) {
			t.Errorf("exchanged secrets %v, want %v", f.secrets, want)
		}
	})

	t.Run("reports a failed mint without leaking the request token", func(t *testing.T) {
		f := newFakeGitHubAndSpacelift(t)
		f.tokenCode = http.StatusForbidden

		_, err := FromEnvironment(context.Background(), f.Client())(f.env(nil))
		assertErrorContains(t, err, "unexpected status 403")
		if strings.Contains(err.Error(), "request-token") {
			t.Errorf("error leaks the request token: %v", err)
		}
		if len(f.secrets) != 0 {
			t.Errorf("exchanged %v after a failed mint", f.secrets)
		}
	})

	for name, tc := range map[string]struct {
		env  map[string]string
		want string
	}{
		"secret and provider both set": {
			env:  map[string]string{EnvSpaceliftAPIKeySecret: "secret"},
			want: "set either SPACELIFT_API_KEY_SECRET or SPACELIFT_API_KEY_OIDC_PROVIDER, not both",
		},
		"audience without provider": {
			env: map[string]string{
				EnvSpaceliftAPIKeyOIDCProvider: "",
				EnvSpaceliftAPIKeyOIDCAudience: "aud",
				EnvSpaceliftAPIKeySecret:       "secret",
			},
			want: "SPACELIFT_API_KEY_OIDC_AUDIENCE is set but SPACELIFT_API_KEY_OIDC_PROVIDER is not",
		},
		"unknown provider": {
			env:  map[string]string{EnvSpaceliftAPIKeyOIDCProvider: "gitlab"},
			want: `unsupported SPACELIFT_API_KEY_OIDC_PROVIDER "gitlab", supported values: github-actions`,
		},
		"GitHub variables missing": {
			env:  map[string]string{envGitHubActionsIDTokenRequestToken: ""},
			want: "permissions: id-token: write",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeGitHubAndSpacelift(t)

			_, err := FromEnvironment(context.Background(), f.Client())(f.env(tc.env))
			assertErrorContains(t, err, tc.want)
			if len(f.secrets) != 0 {
				t.Errorf("exchanged %v, want no exchange", f.secrets)
			}
		})
	}

	t.Run("plain secret keys are unaffected", func(t *testing.T) {
		f := newFakeGitHubAndSpacelift(t)

		_, err := FromEnvironment(context.Background(), f.Client())(f.env(map[string]string{
			EnvSpaceliftAPIKeyOIDCProvider: "",
			EnvSpaceliftAPIKeySecret:       "secret",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []string{"secret"}; !slices.Equal(f.secrets, want) {
			t.Errorf("exchanged secrets %v, want %v", f.secrets, want)
		}
		if len(f.audiences) != 0 {
			t.Errorf("minted a GitHub token for a secret key")
		}
	})
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected an error containing %q, got %v", want, err)
	}
}
