package session

import (
	"context"
	"encoding/base64"
	"net/http"
	"path"
	"strings"
	"testing"
)

// testJWT builds a JWT that FromAPIToken can parse. Only the audience and expiry
// claims are read, and the signature is never verified, so a dummy one will do.
func testJWT(t *testing.T, audience string) string {
	t.Helper()

	enc := func(segment string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(segment))
	}

	return strings.Join([]string{
		enc(`{"alg":"HS256","typ":"JWT"}`),
		enc(`{"aud":"` + audience + `","exp":1516239022}`),
		enc("signature"),
	}, ".")
}

// selectedProfileAlias is the one profile the New tests store; SPACELIFT_PROFILE points at
// it (or deliberately does not) per case.
const selectedProfileAlias = "other"

// newSelectedProfile points SPACELIFT_CONFIG_DIR at a temporary directory holding a single
// selected API token profile. It also clears every credential variable, so a session built
// from the environment can only come from what the caller sets afterwards - not from
// whatever the developer happens to have exported.
func newSelectedProfile(t *testing.T, token string) {
	t.Helper()

	for _, name := range []string{
		EnvSpaceliftAPIToken,
		EnvSpaceliftAPIKeyID,
		EnvSpaceliftAPIKeySecret,
		EnvSpaceliftAPIKeyEndpoint,
		EnvSpaceliftAPIEndpoint,
		EnvSpaceliftAPIGitHubToken,
	} {
		t.Setenv(name, "")
	}

	t.Setenv(EnvSpaceliftConfigDirectory, path.Join(t.TempDir(), "profiles"))

	manager, err := UserProfileManager()
	if err != nil {
		t.Fatalf("could not create profile manager: %v", err)
	}

	if err := manager.Create(&Profile{
		Alias: selectedProfileAlias,
		Credentials: &StoredCredentials{
			Type:        CredentialsTypeAPIToken,
			Endpoint:    "https://spacectl.app.spacelift.io",
			AccessToken: token,
		},
	}); err != nil {
		t.Fatalf("could not create profile %q: %v", selectedProfileAlias, err)
	}
}

func assertToken(t *testing.T, sess Session, want string) {
	t.Helper()

	if sess == nil {
		t.Fatal("expected a session, got nil")
	}

	token, err := sess.BearerToken(context.Background())
	if err != nil {
		t.Fatalf("could not get bearer token: %v", err)
	}

	if token != want {
		t.Fatalf("session was built from the wrong credentials source: got %q, want %q", token, want)
	}
}

func TestNew(t *testing.T) {
	t.Run("prefers the profile named by SPACELIFT_PROFILE over the environment", func(t *testing.T) {
		profileToken := testJWT(t, "profile")
		newSelectedProfile(t, profileToken)

		// Credentials in the environment are valid, but naming a profile is deliberate.
		t.Setenv(EnvSpaceliftAPIToken, testJWT(t, "environment"))
		t.Setenv(EnvSpaceliftProfile, selectedProfileAlias)

		sess, err := New(context.Background(), http.DefaultClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertToken(t, sess, profileToken)
	})

	t.Run("does not fall back to the environment when SPACELIFT_PROFILE is unknown", func(t *testing.T) {
		newSelectedProfile(t, testJWT(t, "profile"))

		t.Setenv(EnvSpaceliftAPIToken, testJWT(t, "environment"))
		t.Setenv(EnvSpaceliftProfile, "typo")

		sess, err := New(context.Background(), http.DefaultClient)
		if err == nil {
			t.Fatalf("expected an error, but a session was built instead: %v", sess)
		}

		if !strings.Contains(err.Error(), EnvSpaceliftProfile) {
			t.Fatalf("error should name %s, got: %v", EnvSpaceliftProfile, err)
		}
	})

	t.Run("uses the environment when SPACELIFT_PROFILE is not set", func(t *testing.T) {
		environmentToken := testJWT(t, "environment")
		newSelectedProfile(t, testJWT(t, "profile"))

		t.Setenv(EnvSpaceliftAPIToken, environmentToken)
		t.Setenv(EnvSpaceliftProfile, "")

		sess, err := New(context.Background(), http.DefaultClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertToken(t, sess, environmentToken)
	})

	t.Run("falls back to the selected profile when the environment has no credentials", func(t *testing.T) {
		profileToken := testJWT(t, "profile")
		newSelectedProfile(t, profileToken)

		t.Setenv(EnvSpaceliftAPIToken, "")
		t.Setenv(EnvSpaceliftProfile, "")

		sess, err := New(context.Background(), http.DefaultClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertToken(t, sess, profileToken)
	})
}
