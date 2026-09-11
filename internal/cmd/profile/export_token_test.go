package profile

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/spacelift-io/spacectl/client/session"
)

// sampleAPIToken is a JWT with header {"alg":"HS256","typ":"JWT"} and payload
// {"aud":"spacectl","exp":1516239022}, generated at https://jwt.io. It only
// needs to carry a single audience claim to be parseable by FromAPIToken.
const sampleAPIToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJzcGFjZWN0bCIsImV4cCI6MTUxNjIzOTAyMn0.fsKd_N2TKXpx83JSPPw47zYzQ8sbSzGVPZcyGpwp05U" //nolint:gosec // sample JWT for tests, not a real credential

// envLookup builds an os.LookupEnv-style func from a map.
func envLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}

func TestResolveSession(t *testing.T) {
	// resolveSession rejects an unresolvable SPACELIFT_PROFILE, so a value left in the
	// developer's own shell would fail the cases below. Subtests that want an override set
	// it themselves after this.
	t.Setenv(session.EnvSpaceliftProfile, "")

	t.Run("falls back to the environment when no profile manager is set", func(t *testing.T) {
		server := newExchangeServer(t, "EnvJWT")
		defer server.Close()

		lookup := envLookup(map[string]string{
			session.EnvSpaceliftAPIKeyEndpoint: server.URL,
			session.EnvSpaceliftAPIKeyID:       "key-id",
			session.EnvSpaceliftAPIKeySecret:   "oidc-token",
		})

		sess, err := resolveSession(context.Background(), nil, server.Client(), lookup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertBearerToken(t, sess, "EnvJWT")
	})

	t.Run("falls back to the environment when no profile is selected", func(t *testing.T) {
		server := newExchangeServer(t, "EnvJWT")
		defer server.Close()

		// A manager pointing at an empty directory has no current profile,
		// mirroring a CI run that authenticates purely through the environment.
		manager, err := session.NewProfileManager(path.Join(t.TempDir(), "profiles"))
		if err != nil {
			t.Fatalf("could not create profile manager: %v", err)
		}

		lookup := envLookup(map[string]string{
			session.EnvSpaceliftAPIKeyEndpoint: server.URL,
			session.EnvSpaceliftAPIKeyID:       "key-id",
			session.EnvSpaceliftAPIKeySecret:   "oidc-token",
		})

		sess, err := resolveSession(context.Background(), manager, server.Client(), lookup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertBearerToken(t, sess, "EnvJWT")
	})

	t.Run("prefers the selected profile over the environment", func(t *testing.T) {
		manager, err := session.NewProfileManager(path.Join(t.TempDir(), "profiles"))
		if err != nil {
			t.Fatalf("could not create profile manager: %v", err)
		}

		if err := manager.Create(&session.Profile{
			Alias: "default",
			Credentials: &session.StoredCredentials{
				Type:        session.CredentialsTypeAPIToken,
				Endpoint:    "https://spacectl.app.spacelift.io",
				AccessToken: sampleAPIToken,
			},
		}); err != nil {
			t.Fatalf("could not create profile: %v", err)
		}

		// Environment points somewhere that would explode if it were used,
		// proving the profile takes precedence.
		lookup := envLookup(map[string]string{
			session.EnvSpaceliftAPIKeyEndpoint: "http://127.0.0.1:0",
			session.EnvSpaceliftAPIKeyID:       "key-id",
			session.EnvSpaceliftAPIKeySecret:   "oidc-token",
		})

		sess, err := resolveSession(context.Background(), manager, http.DefaultClient, lookup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertBearerToken(t, sess, sampleAPIToken)
	})

	t.Run("uses the profile named by SPACELIFT_PROFILE", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "override")

		manager := managerWithAPITokenProfiles(t, "selected", "override")

		lookup := envLookup(map[string]string{
			session.EnvSpaceliftAPIKeyEndpoint: "http://127.0.0.1:0",
			session.EnvSpaceliftAPIKeyID:       "key-id",
			session.EnvSpaceliftAPIKeySecret:   "oidc-token",
		})

		sess, err := resolveSession(context.Background(), manager, http.DefaultClient, lookup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Each profile carries a token named after its alias, so the bearer token
		// says which profile the session was actually built from.
		assertBearerToken(t, sess, aliasToken("override"))
	})

	t.Run("does not fall back to the environment when SPACELIFT_PROFILE is unknown", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "typo")

		manager := managerWithAPITokenProfiles(t, "selected")

		// Valid credentials in the environment must not rescue a bad alias: exporting a
		// token for a different account is worse than failing.
		server := newExchangeServer(t, "EnvJWT")
		defer server.Close()

		lookup := envLookup(map[string]string{
			session.EnvSpaceliftAPIKeyEndpoint: server.URL,
			session.EnvSpaceliftAPIKeyID:       "key-id",
			session.EnvSpaceliftAPIKeySecret:   "oidc-token",
		})

		sess, err := resolveSession(context.Background(), manager, server.Client(), lookup)
		if err == nil {
			t.Fatalf("expected an error, but a session was built instead: %v", sess)
		}

		if !strings.Contains(err.Error(), session.EnvSpaceliftProfile) {
			t.Fatalf("error should name %s, got: %v", session.EnvSpaceliftProfile, err)
		}
	})
}

// aliasToken returns a JWT unique to a profile alias, so that assertions can tell
// which profile a session came from.
func aliasToken(alias string) string {
	enc := func(segment string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(segment))
	}

	return strings.Join([]string{
		enc(`{"alg":"HS256","typ":"JWT"}`),
		enc(`{"aud":"` + alias + `","exp":1516239022}`),
		enc("signature"),
	}, ".")
}

// managerWithAPITokenProfiles creates an API token profile per alias in a temporary
// directory. The first alias ends up selected, mirroring `spacectl profile select`.
func managerWithAPITokenProfiles(t *testing.T, aliases ...string) *session.ProfileManager {
	t.Helper()

	manager, err := session.NewProfileManager(path.Join(t.TempDir(), "profiles"))
	if err != nil {
		t.Fatalf("could not create profile manager: %v", err)
	}

	for _, alias := range aliases {
		if err := manager.Create(&session.Profile{
			Alias: alias,
			Credentials: &session.StoredCredentials{
				Type:        session.CredentialsTypeAPIToken,
				Endpoint:    "https://spacectl.app.spacelift.io",
				AccessToken: aliasToken(alias),
			},
		}); err != nil {
			t.Fatalf("could not create profile %q: %v", alias, err)
		}
	}

	if err := manager.Select(aliases[0]); err != nil {
		t.Fatalf("could not select profile %q: %v", aliases[0], err)
	}

	return manager
}

// newExchangeServer returns a mock GraphQL server that answers the apiKeyUser
// mutation with the given JWT.
func newExchangeServer(t *testing.T, jwt string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		if _, err := rw.Write([]byte(`{"data":{"apiKeyUser":{"jwt":"` + jwt + `","validUntil":4102444800}}}`)); err != nil {
			t.Errorf("could not write mock response: %v", err)
		}
	}))
}

func assertBearerToken(t *testing.T, sess session.Session, want string) {
	t.Helper()
	if sess == nil {
		t.Fatal("expected a session, got nil")
	}
	token, err := sess.BearerToken(context.Background())
	if err != nil {
		t.Fatalf("could not get bearer token: %v", err)
	}
	if token != want {
		t.Fatalf("unexpected bearer token: got %q, want %q", token, want)
	}
}
