package profile

import (
	"testing"

	"github.com/spacelift-io/spacectl/client/session"
)

// TestPersistAccessCredentialsKeepsProxyAuth covers re-authenticating a profile that is
// configured to authenticate through a reverse proxy. Logging in produces fresh Spacelift
// credentials, but proxy_auth is configuration rather than a credential, and losing it on
// every browser login would make the profile unusable until config.json is edited again.
func TestPersistAccessCredentialsKeepsProxyAuth(t *testing.T) {
	proxyAuth := &session.ProxyAuthConfig{Command: []string{"/usr/local/bin/proxy-auth", "--cached"}}

	configure := func(t *testing.T, alias string) {
		t.Helper()

		newConfigDir(t, alias)
		newManager(t)

		existing, err := manager.Get(alias)
		if err != nil {
			t.Fatalf("could not read profile: %v", err)
		}
		existing.Credentials.ProxyAuth = proxyAuth
		if err := manager.Create(existing); err != nil {
			t.Fatalf("could not store proxy auth: %v", err)
		}
	}

	login := func(t *testing.T, alias string, creds *session.StoredCredentials) *session.Profile {
		t.Helper()

		profileAlias = alias
		aliasFromArgs = true
		t.Cleanup(func() { profileAlias = ""; aliasFromArgs = false })

		if err := persistAccessCredentials(creds); err != nil {
			t.Fatalf("could not persist credentials: %v", err)
		}

		reloaded, err := session.UserProfileManager()
		if err != nil {
			t.Fatalf("could not reload profile manager: %v", err)
		}
		profile, err := reloaded.Get(alias)
		if err != nil {
			t.Fatalf("could not read profile: %v", err)
		}
		return profile
	}

	t.Run("re-authenticating keeps the existing proxy auth", func(t *testing.T) {
		configure(t, "proxied")

		profile := login(t, "proxied", &session.StoredCredentials{
			Type:        session.CredentialsTypeAPIToken,
			Endpoint:    "https://spacectl.app.spacelift.io",
			AccessToken: sampleAPIToken,
		})

		if profile.Credentials.AccessToken != sampleAPIToken {
			t.Fatalf("the new token was not stored")
		}
		if got := profile.Credentials.ProxyAuth; got == nil || len(got.Command) != 2 || got.Command[0] != proxyAuth.Command[0] {
			t.Fatalf("proxy auth was not preserved across login: got %+v", got)
		}
	})

	t.Run("a new profile does not inherit proxy auth from another", func(t *testing.T) {
		configure(t, "proxied")

		profile := login(t, "plain", &session.StoredCredentials{
			Type:        session.CredentialsTypeAPIToken,
			Endpoint:    "https://spacectl.app.spacelift.io",
			AccessToken: sampleAPIToken,
		})

		if profile.Credentials.ProxyAuth != nil {
			t.Fatalf("a brand new profile should have no proxy auth, got %+v", profile.Credentials.ProxyAuth)
		}
	})
}
