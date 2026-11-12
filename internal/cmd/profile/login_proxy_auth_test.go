package profile

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client/session"
)

func TestLoginProxyAuthFlags(t *testing.T) {
	for _, method := range []string{methodAPI, methodGithub} {
		for _, source := range []string{"command", "token_file"} {
			t.Run(method+"/"+source, func(t *testing.T) {
				setupProxyAuthLogin(t)
				proxyAuth := newLoginProxyAuth(t, source)
				server := newProxyAuthLoginServer(t, method, "Bearer proxy-token", http.StatusOK)

				require.NoError(t, runProxyAuthLogin(server.URL, method, proxyAuth))
				require.Equal(t, 1, server.requests)
				assertSavedProxyAuth(t, proxyAuth)
			})
		}
	}
}

func TestLoginProxyAuthReplacesExistingSettings(t *testing.T) {
	for _, test := range []struct {
		source   string
		previous *session.ProxyAuthConfig
	}{
		{source: "command", previous: &session.ProxyAuthConfig{TokenFile: filepath.Join(t.TempDir(), "missing-token")}},
		{source: "token_file", previous: &session.ProxyAuthConfig{Command: []string{filepath.Join(t.TempDir(), "missing-helper")}}},
	} {
		t.Run(test.source, func(t *testing.T) {
			setupProxyAuthLogin(t)
			proxyAuth := newLoginProxyAuth(t, test.source)
			server := newProxyAuthLoginServer(t, methodAPI, "Bearer proxy-token", http.StatusOK)
			createProxyAuthLoginProfile(t, server.URL, test.previous)

			require.NoError(t, runProxyAuthLogin(server.URL, methodAPI, proxyAuth))
			require.Equal(t, 1, server.requests)
			assertSavedProxyAuth(t, proxyAuth)
		})
	}
}

func TestLoginProxyAuthPreservesExistingSettings(t *testing.T) {
	for _, source := range []string{"command", "token_file"} {
		t.Run(source, func(t *testing.T) {
			setupProxyAuthLogin(t)
			proxyAuth := newLoginProxyAuth(t, source)
			server := newProxyAuthLoginServer(t, methodAPI, "Bearer proxy-token", http.StatusOK)
			createProxyAuthLoginProfile(t, server.URL, proxyAuth)

			require.NoError(t, runProxyAuthLogin(server.URL, methodAPI, nil))
			require.Equal(t, 1, server.requests)
			assertSavedProxyAuth(t, proxyAuth)
		})
	}
}

func TestLoginProxyAuthRejectsUntrustedInheritedCommand(t *testing.T) {
	for _, configFile := range []string{"writable", "symlink"} {
		t.Run(configFile, func(t *testing.T) {
			setupProxyAuthLogin(t)
			proxyAuth := newLoginProxyAuth(t, "command")
			server := newProxyAuthLoginServer(t, methodAPI, "Bearer proxy-token", http.StatusOK)
			createProxyAuthLoginProfile(t, server.URL, proxyAuth)
			originalConfig := readProxyAuthLoginConfig(t)
			wantError := makeProxyAuthLoginConfigUntrusted(t, configFile)

			err := runProxyAuthLogin(server.URL, methodAPI, nil)
			require.ErrorContains(t, err, wantError)
			require.Zero(t, server.requests)
			require.Equal(t, originalConfig, readProxyAuthLoginConfig(t), "rejected proxy authentication changed the profile configuration")
		})
	}
}

func TestLoginProxyAuthExplicitCommandIgnoresUntrustedConfig(t *testing.T) {
	for _, configFile := range []string{"writable", "symlink"} {
		t.Run(configFile, func(t *testing.T) {
			setupProxyAuthLogin(t)
			proxyAuth := newLoginProxyAuth(t, "command")
			server := newProxyAuthLoginServer(t, methodAPI, "Bearer proxy-token", http.StatusOK)
			createProxyAuthLoginProfile(t, server.URL, &session.ProxyAuthConfig{
				TokenFile: filepath.Join(t.TempDir(), "missing-token"),
			})
			makeProxyAuthLoginConfigUntrusted(t, configFile)

			require.NoError(t, runProxyAuthLogin(server.URL, methodAPI, proxyAuth))
			require.Equal(t, 1, server.requests)
			assertSavedProxyAuth(t, proxyAuth)
		})
	}
}

func TestLoginProxyAuthDoesNotInheritSelectedProfile(t *testing.T) {
	setupProxyAuthLogin(t)
	server := newProxyAuthLoginServer(t, methodAPI, "", http.StatusOK)

	require.NoError(t, runProxyAuthLogin(server.URL, methodAPI, nil))
	require.Equal(t, 1, server.requests)
	assertSavedProxyAuth(t, nil)
}

func TestLoginProxyAuthFailedLoginDoesNotSaveSettings(t *testing.T) {
	for _, source := range []string{"command", "token_file"} {
		t.Run(source, func(t *testing.T) {
			setupProxyAuthLogin(t)
			proxyAuth := newLoginProxyAuth(t, source)
			server := newProxyAuthLoginServer(t, methodAPI, "Bearer proxy-token", http.StatusForbidden)
			originalConfig := readProxyAuthLoginConfig(t)

			require.Error(t, runProxyAuthLogin(server.URL, methodAPI, proxyAuth))
			require.Equal(t, 1, server.requests)
			require.Equal(t, originalConfig, readProxyAuthLoginConfig(t), "failed login changed the profile configuration")
		})
	}
}

func setupProxyAuthLogin(t *testing.T) {
	t.Helper()

	t.Setenv(session.EnvSpaceliftProfile, "")
	t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
	t.Setenv(session.EnvSpaceliftAPIKeyID, "key-id")
	t.Setenv(session.EnvSpaceliftAPIKeySecret, "key-secret")
	t.Setenv(session.EnvSpaceliftAPIGitHubToken, "github-token")
	newConfigDir(t, "selected")
	newManager(t)
	apiTokenProfile = nil
	t.Cleanup(func() { apiTokenProfile = nil; profileAlias = ""; aliasFromArgs = false })

	// The selected profile must not supply proxy credentials for a login to target.
	selected, err := manager.Get("selected")
	require.NoError(t, err)
	selected.Credentials.ProxyAuth = &session.ProxyAuthConfig{TokenFile: filepath.Join(t.TempDir(), "missing")}
	require.NoError(t, manager.Create(selected))
}

func newLoginProxyAuth(t *testing.T, source string) *session.ProxyAuthConfig {
	t.Helper()

	switch source {
	case "command":
		if runtime.GOOS == "windows" {
			t.Skip("test helper uses /bin/sh")
		}
		script := filepath.Join(t.TempDir(), "proxy-auth.sh")
		require.NoError(t, os.WriteFile(script, []byte(`if [ "$#" -ne 3 ] || [ "$1" != "--audience=a,b" ] || [ "$2" != "value with spaces" ] || [ "$3" != "" ]; then
  exit 1
fi
printf 'proxy-token\n'
`), 0o600))
		return &session.ProxyAuthConfig{Command: []string{"/bin/sh", script, "--audience=a,b", "value with spaces", ""}}
	case "token_file":
		tokenFile := filepath.Join(t.TempDir(), "proxy token.txt")
		require.NoError(t, os.WriteFile(tokenFile, []byte("  proxy-token\n"), 0o600))
		return &session.ProxyAuthConfig{TokenFile: tokenFile}
	default:
		t.Fatalf("unknown proxy authentication source %q", source)
		return nil
	}
}

type proxyAuthLoginServer struct {
	*httptest.Server
	requests int
}

func newProxyAuthLoginServer(t *testing.T, method, expectedHeader string, status int) *proxyAuthLoginServer {
	t.Helper()

	server := &proxyAuthLoginServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.requests++
		if got := r.Header.Get("Proxy-Authorization"); got != expectedHeader {
			t.Errorf("Proxy-Authorization = %q, want %q", got, expectedHeader)
		}
		if status != http.StatusOK {
			http.Error(w, "login rejected", status)
			return
		}
		field := "apiKeyUser"
		if method == methodGithub {
			field = "oauthUser"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"%s":{"jwt":"token","validUntil":4102444800}}}`, field)
	}))
	t.Cleanup(server.Close)
	return server
}

func createProxyAuthLoginProfile(t *testing.T, endpoint string, proxyAuth *session.ProxyAuthConfig) {
	t.Helper()

	require.NoError(t, manager.Create(&session.Profile{
		Alias: "target",
		Credentials: &session.StoredCredentials{
			Type: session.CredentialsTypeAPIToken, Endpoint: endpoint, ProxyAuth: proxyAuth,
		},
	}))
	require.NoError(t, manager.Select("selected"))
}

func runProxyAuthLogin(endpoint, method string, proxyAuth *session.ProxyAuthConfig) error {
	args := []string{"profile", "login", "--method", method, "--endpoint", endpoint}
	if proxyAuth != nil {
		for _, arg := range proxyAuth.Command {
			args = append(args, "--proxy-auth-command="+arg)
		}
		if proxyAuth.TokenFile != "" {
			args = append(args, "--proxy-auth-token-file", proxyAuth.TokenFile)
		}
	}
	args = append(args, "target")
	return newProfileCommand().Run(context.Background(), args)
}

func assertSavedProxyAuth(t *testing.T, expected *session.ProxyAuthConfig) {
	t.Helper()

	reloaded, err := session.UserProfileManager()
	require.NoError(t, err)
	profile, err := reloaded.Get("target")
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.Equal(t, expected, profile.Credentials.ProxyAuth)
}

func readProxyAuthLoginConfig(t *testing.T) []byte {
	t.Helper()

	config, err := os.ReadFile(manager.ConfigurationFile)
	require.NoError(t, err)
	return config
}

func makeProxyAuthLoginConfigUntrusted(t *testing.T, kind string) string {
	t.Helper()

	switch kind {
	case "writable":
		require.NoError(t, os.Chmod(manager.ConfigurationFile, 0o666))
		return "writable by other users"
	case "symlink":
		target := manager.ConfigurationFile + ".target"
		require.NoError(t, os.Rename(manager.ConfigurationFile, target))
		require.NoError(t, os.Symlink(target, manager.ConfigurationFile))
		return "symbolic link"
	default:
		t.Fatalf("unknown config file kind %q", kind)
		return ""
	}
}

func TestLoginProxyAuthCommandRejectsInvalidPath(t *testing.T) {
	for _, command := range []string{"", "proxy-auth", "./proxy-auth"} {
		t.Run(command, func(t *testing.T) {
			newConfigDir(t, "selected")
			err := runLoginBefore(t, "--proxy-auth-command="+command, "target")
			if err == nil || !strings.Contains(err.Error(), "absolute executable path") {
				t.Fatalf("expected an absolute executable path error, got %v", err)
			}
		})
	}
}

func TestLoginProxyAuthTokenFileRejectsInvalidPath(t *testing.T) {
	for _, path := range []string{"", "proxy-token", "./proxy-token"} {
		t.Run(path, func(t *testing.T) {
			newConfigDir(t, "selected")
			err := runLoginBefore(t, "--proxy-auth-token-file="+path, "target")
			if err == nil || !strings.Contains(err.Error(), "absolute file path") {
				t.Fatalf("expected an absolute file path error, got %v", err)
			}
		})
	}
}

func TestLoginProxyAuthFlagsAreMutuallyExclusive(t *testing.T) {
	commandFlag := "--proxy-auth-command=" + filepath.Join(t.TempDir(), "proxy-auth")
	tokenFileFlag := "--proxy-auth-token-file=" + filepath.Join(t.TempDir(), "proxy-token")
	for _, args := range [][]string{
		{commandFlag, tokenFileFlag, "target"},
		{tokenFileFlag, commandFlag, "target"},
	} {
		newConfigDir(t, "selected")
		err := runLoginBefore(t, args...)
		if err == nil || !strings.Contains(err.Error(), "option proxy-auth-command cannot be set along with option proxy-auth-token-file") {
			t.Fatalf("expected mutually exclusive proxy auth flags to fail, got %v", err)
		}
	}
}

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
