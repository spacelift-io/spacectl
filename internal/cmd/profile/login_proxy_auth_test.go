package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spacelift-io/spacectl/client/session"
)

func TestLoginProxyAuth(t *testing.T) {
	for _, test := range []struct {
		name       string
		method     string
		existing   bool
		source     string
		preserve   bool
		loginFails bool
		configFile string
		trustError string
	}{
		{name: "new API key profile", method: methodAPI, source: "command"},
		{name: "new GitHub profile", method: methodGithub, source: "command"},
		{name: "replaces existing token file", method: methodAPI, existing: true, source: "command"},
		{name: "preserves existing command", method: methodAPI, existing: true, source: "command", preserve: true},
		{name: "rejects inherited command from writable config", method: methodAPI, existing: true, source: "command", preserve: true, configFile: "writable", trustError: "writable by other users"},
		{name: "rejects inherited command from symlinked config", method: methodAPI, existing: true, source: "command", preserve: true, configFile: "symlink", trustError: "symbolic link"},
		{name: "explicit command ignores writable config", method: methodAPI, existing: true, source: "command", configFile: "writable"},
		{name: "explicit command ignores symlinked config", method: methodAPI, existing: true, source: "command", configFile: "symlink"},
		{name: "does not inherit selected profile settings", method: methodAPI},
		{name: "failed login does not save command", method: methodAPI, source: "command", loginFails: true},
		{name: "new API key profile with token file", method: methodAPI, source: "token_file"},
		{name: "new GitHub profile with token file", method: methodGithub, source: "token_file"},
		{name: "token file replaces existing command", method: methodAPI, existing: true, source: "token_file"},
		{name: "preserves existing token file", method: methodAPI, existing: true, source: "token_file", preserve: true},
		{name: "failed login does not save token file", method: methodAPI, source: "token_file", loginFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.source == "command" && runtime.GOOS == "windows" {
				t.Skip("test helper uses /bin/sh")
			}

			t.Setenv(session.EnvSpaceliftProfile, "")
			t.Setenv(session.EnvSpaceliftProxyAuthToken, "")
			t.Setenv(session.EnvSpaceliftAPIKeyID, "key-id")
			t.Setenv(session.EnvSpaceliftAPIKeySecret, "key-secret")
			t.Setenv(session.EnvSpaceliftAPIGitHubToken, "github-token")
			newConfigDir(t, "selected")
			newManager(t)
			apiTokenProfile = nil
			t.Cleanup(func() { apiTokenProfile = nil; profileAlias = ""; aliasFromArgs = false })

			// The selected profile must not supply proxy credentials for this login.
			selected, err := manager.Get("selected")
			if err != nil {
				t.Fatal(err)
			}
			selected.Credentials.ProxyAuth = &session.ProxyAuthConfig{TokenFile: filepath.Join(t.TempDir(), "missing")}
			if err := manager.Create(selected); err != nil {
				t.Fatal(err)
			}

			script := filepath.Join(t.TempDir(), "proxy-auth.sh")
			if err := os.WriteFile(script, []byte(`if [ "$#" -ne 3 ] || [ "$1" != "--audience=a,b" ] || [ "$2" != "value with spaces" ] || [ "$3" != "" ]; then
  exit 1
fi
printf 'proxy-token\n'
`), 0o600); err != nil {
				t.Fatal(err)
			}
			command := []string{"/bin/sh", script, "--audience=a,b", "value with spaces", ""}
			tokenFile := filepath.Join(t.TempDir(), "proxy token.txt")
			if err := os.WriteFile(tokenFile, []byte("  proxy-token\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var expectedProxyAuth *session.ProxyAuthConfig
			switch test.source {
			case "command":
				expectedProxyAuth = &session.ProxyAuthConfig{Command: command}
			case "token_file":
				expectedProxyAuth = &session.ProxyAuthConfig{TokenFile: tokenFile}
			}

			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var expectedHeader string
				if expectedProxyAuth != nil {
					expectedHeader = "Bearer proxy-token"
				}
				if got := r.Header.Get("Proxy-Authorization"); got != expectedHeader {
					t.Errorf("Proxy-Authorization = %q, want %q", got, expectedHeader)
				}
				if test.loginFails {
					http.Error(w, "login rejected", http.StatusForbidden)
					return
				}
				field := "apiKeyUser"
				if test.method == methodGithub {
					field = "oauthUser"
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"data":{"%s":{"jwt":"token","validUntil":4102444800}}}`, field)
			}))
			defer server.Close()

			if test.existing {
				proxyAuth := selected.Credentials.ProxyAuth
				if test.source == "token_file" {
					proxyAuth = &session.ProxyAuthConfig{Command: []string{filepath.Join(t.TempDir(), "missing-helper")}}
				}
				if test.preserve {
					proxyAuth = expectedProxyAuth
				}
				if err := manager.Create(&session.Profile{
					Alias: "target",
					Credentials: &session.StoredCredentials{
						Type: session.CredentialsTypeAPIToken, Endpoint: server.URL, ProxyAuth: proxyAuth,
					},
				}); err != nil {
					t.Fatal(err)
				}
				if err := manager.Select("selected"); err != nil {
					t.Fatal(err)
				}
			}
			originalConfig, err := os.ReadFile(manager.ConfigurationFile)
			if err != nil {
				t.Fatal(err)
			}
			switch test.configFile {
			case "writable":
				if err := os.Chmod(manager.ConfigurationFile, 0o666); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				target := manager.ConfigurationFile + ".target"
				if err := os.Rename(manager.ConfigurationFile, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, manager.ConfigurationFile); err != nil {
					t.Fatal(err)
				}
			}

			args := []string{"profile", "login", "--method", test.method, "--endpoint", server.URL}
			if !test.preserve {
				switch test.source {
				case "command":
					for _, arg := range command {
						args = append(args, "--proxy-auth-command="+arg)
					}
				case "token_file":
					args = append(args, "--proxy-auth-token-file", tokenFile)
				}
			}
			args = append(args, "target")
			err = newProfileCommand().Run(context.Background(), args)
			if test.trustError != "" {
				if err == nil || !strings.Contains(err.Error(), test.trustError) {
					t.Fatalf("login error = %v, want %q", err, test.trustError)
				}
				if requests != 0 {
					t.Errorf("made %d login requests, want 0", requests)
				}
				config, err := os.ReadFile(manager.ConfigurationFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(config) != string(originalConfig) {
					t.Fatal("rejected proxy authentication changed the profile configuration")
				}
				return
			}
			if test.loginFails {
				if err == nil {
					t.Fatal("expected login to fail")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Errorf("made %d login requests, want 1", requests)
			}

			reloaded, err := session.UserProfileManager()
			if err != nil {
				t.Fatal(err)
			}
			profile, err := reloaded.Get("target")
			if err != nil {
				t.Fatal(err)
			}
			if test.loginFails {
				if profile != nil || reloaded.Configuration.CurrentProfileAlias != "selected" {
					t.Fatal("failed login changed the profile configuration")
				}
				return
			}
			if profile == nil || !reflect.DeepEqual(profile.Credentials.ProxyAuth, expectedProxyAuth) {
				encoded, _ := json.Marshal(profile)
				t.Fatalf("unexpected saved proxy authentication: %s", encoded)
			}
		})
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
