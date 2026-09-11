package profile

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
)

// newConfigDir points SPACELIFT_CONFIG_DIR at a temporary directory holding one API token
// profile per alias, the first of which is selected.
func newConfigDir(t *testing.T, aliases ...string) {
	t.Helper()

	dir := path.Join(t.TempDir(), "profiles")
	t.Setenv(session.EnvSpaceliftConfigDirectory, dir)

	manager, err := session.NewProfileManager(dir)
	if err != nil {
		t.Fatalf("could not create profile manager: %v", err)
	}

	for _, alias := range aliases {
		if err := manager.Create(&session.Profile{
			Alias: alias,
			Credentials: &session.StoredCredentials{
				Type:        session.CredentialsTypeAPIToken,
				Endpoint:    "https://spacectl.app.spacelift.io",
				AccessToken: sampleAPIToken,
			},
		}); err != nil {
			t.Fatalf("could not create profile %q: %v", alias, err)
		}
	}

	if err := manager.Select(aliases[0]); err != nil {
		t.Fatalf("could not select profile %q: %v", aliases[0], err)
	}
}

// newManager sets the package-level manager the commands use, the way the profile subtree's
// Before hook does. Call it after SPACELIFT_PROFILE is set: the manager captures the override
// at construction, so the order matters.
func newManager(t *testing.T) {
	t.Helper()

	var err error
	if manager, err = session.UserProfileManager(); err != nil {
		t.Fatalf("could not create profile manager: %v", err)
	}
}

// captureStdout runs fn with os.Stdout redirected, and returns what it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = writer

	fn()

	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close pipe: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("could not read pipe: %v", err)
	}

	return buf.String()
}

// runLoginBefore runs `profile login` through the real command tree - so the profile
// subtree's own Before hook runs too - far enough to exercise login's Before hook, with the
// action stubbed out so the interactive flow never starts.
func runLoginBefore(t *testing.T, args ...string) error {
	t.Helper()

	parent := Command()
	for _, subcommand := range parent.Commands {
		if subcommand.Name == "login" {
			subcommand.Action = func(context.Context, *cli.Command) error { return nil }
		}
	}

	return parent.Run(context.Background(), append([]string{"profile", "login"}, args...))
}

// TestProfileOverrideIsValidated covers the ValidateProfileOverride guards on the commands
// that resolve the selected profile. An alias naming no profile must stop them with a message
// naming the variable, rather than being swallowed as "nothing is selected" - while still
// leaving `profile login <alias>` able to create that very profile.
func TestProfileOverrideIsValidated(t *testing.T) {
	for _, subcommand := range []string{"current", "list"} {
		t.Run(subcommand+" rejects an unknown SPACELIFT_PROFILE", func(t *testing.T) {
			newConfigDir(t, "selected")
			t.Setenv(session.EnvSpaceliftProfile, "typo")

			err := Command().Run(context.Background(), []string{"profile", subcommand})
			if err == nil {
				t.Fatal("expected an error, got none")
			}

			if !strings.Contains(err.Error(), session.EnvSpaceliftProfile) {
				t.Fatalf("error should name %s, got: %v", session.EnvSpaceliftProfile, err)
			}
			if !strings.Contains(err.Error(), "typo") {
				t.Fatalf("error should name the bad alias, got: %v", err)
			}
		})

		t.Run(subcommand+" accepts a known SPACELIFT_PROFILE", func(t *testing.T) {
			newConfigDir(t, "selected", "other")
			t.Setenv(session.EnvSpaceliftProfile, "other")

			if err := Command().Run(context.Background(), []string{"profile", subcommand}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

	t.Run("login can still create the profile the override names", func(t *testing.T) {
		// Validating the override for the whole subtree would make this bootstrap
		// impossible, and the error would tell the user to run the command that just
		// failed. login only needs the override to resolve when no alias is given.
		newConfigDir(t, "selected")
		t.Setenv(session.EnvSpaceliftProfile, "brandnew")
		t.Cleanup(func() { profileAlias = "" })

		// The Before hook is what validates; stub the Action so the test does not enter
		// the interactive login flow.
		if err := runLoginBefore(t, "brandnew"); err != nil {
			t.Fatalf("naming an alias explicitly should not require the override to exist: %v", err)
		}
	})

	t.Run("login with no alias rejects an unknown override", func(t *testing.T) {
		newConfigDir(t, "selected")
		t.Setenv(session.EnvSpaceliftProfile, "typo")
		t.Cleanup(func() { profileAlias = "" })

		err := runLoginBefore(t)
		if err == nil {
			t.Fatal("expected an error, got none")
		}
		if !strings.Contains(err.Error(), session.EnvSpaceliftProfile) {
			t.Fatalf("error should name %s, got: %v", session.EnvSpaceliftProfile, err)
		}
	})

	t.Run("current reports the overridden profile", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "other")

		out := captureStdout(t, func() {
			if err := Command().Run(context.Background(), []string{"profile", "current"}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})

		if strings.TrimSpace(out) != "other" {
			t.Fatalf("expected the override alias, got %q", strings.TrimSpace(out))
		}
	})
}

// TestPersistAccessCredentialsSelection covers which logins change the stored selection.
// Re-authenticating the profile SPACELIFT_PROFILE names must leave it alone, because the
// variable is per-shell; an alias the user typed must still select, because that is a
// deliberate choice the variable should not swallow.
func TestPersistAccessCredentialsSelection(t *testing.T) {
	login := func(t *testing.T, alias string, fromArgs bool) string {
		t.Helper()

		profileAlias = alias
		aliasFromArgs = fromArgs
		t.Cleanup(func() { profileAlias = ""; aliasFromArgs = false })

		if err := persistAccessCredentials(&session.StoredCredentials{
			Type:        session.CredentialsTypeAPIToken,
			Endpoint:    "https://spacectl.app.spacelift.io",
			AccessToken: sampleAPIToken,
		}); err != nil {
			t.Fatalf("could not persist credentials: %v", err)
		}

		reloaded, err := session.UserProfileManager()
		if err != nil {
			t.Fatalf("could not reload profile manager: %v", err)
		}

		return reloaded.Configuration.CurrentProfileAlias
	}

	t.Run("re-authenticating the override does not change the stored selection", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "other")
		newManager(t)

		if got := login(t, "other", false); got != "selected" {
			t.Fatalf("the override leaked into the stored selection: got %q, want %q", got, "selected")
		}
	})

	t.Run("an alias typed on the command line still selects", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "other")
		newManager(t)

		// `spacectl profile login other` while SPACELIFT_PROFILE=other: the user named it,
		// so it must take effect rather than being mistaken for a re-auth.
		if got := login(t, "other", true); got != "other" {
			t.Fatalf("an explicitly named alias was not selected: got %q, want %q", got, "other")
		}
	})

	t.Run("logging in without an override selects as usual", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "")
		newManager(t)

		if got := login(t, "other", true); got != "other" {
			t.Fatalf("expected the login to select the profile: got %q", got)
		}
	})
}

// TestSelectWarnsWhenOverridden covers the warning `profile select` prints when the stored
// selection it just wrote will be shadowed by SPACELIFT_PROFILE.
func TestSelectWarnsWhenOverridden(t *testing.T) {
	const warning = "takes precedence over the profile you just selected"

	selectProfile := func(t *testing.T, alias string) string {
		t.Helper()

		return captureStdout(t, func() {
			if err := Command().Run(context.Background(), []string{"profile", "select", alias}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("warns when the override names a different profile", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "other")
		t.Cleanup(func() { profileAlias = "" })

		out := selectProfile(t, "selected")

		if !strings.Contains(out, warning) {
			t.Fatalf("expected the override warning, got %q", out)
		}

		// The selection is still written - the warning says it will not take effect in
		// this shell, not that it was refused.
		reloaded, err := session.UserProfileManager()
		if err != nil {
			t.Fatalf("could not reload profile manager: %v", err)
		}
		if got := reloaded.Configuration.CurrentProfileAlias; got != "selected" {
			t.Fatalf("expected the selection to be persisted, got %q", got)
		}
	})

	t.Run("stays silent when the override names the profile being selected", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "other")
		t.Cleanup(func() { profileAlias = "" })

		if out := selectProfile(t, "other"); strings.Contains(out, warning) {
			t.Fatalf("expected no warning when selecting the overridden profile, got %q", out)
		}
	})

	t.Run("stays silent when no override is set", func(t *testing.T) {
		newConfigDir(t, "selected", "other")
		t.Setenv(session.EnvSpaceliftProfile, "")
		t.Cleanup(func() { profileAlias = "" })

		if out := selectProfile(t, "other"); strings.Contains(out, warning) {
			t.Fatalf("expected no warning without an override, got %q", out)
		}
	})
}

// TestPrintEnvWarning covers the two messages login prints about SPACELIFT_PROFILE: the
// override warning must not fire for the profile being logged into, and the usual
// "environment takes precedence" advice is backwards while an override is set.
func TestPrintEnvWarning(t *testing.T) {
	const overrideWarning = "takes precedence over the profile selected by logging in"

	t.Run("warns when the override names a different profile", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "other")
		profileAlias = "selected"
		t.Cleanup(func() { profileAlias = "" })

		out := captureStdout(t, printEnvWarning)

		if !strings.Contains(out, overrideWarning) {
			t.Fatalf("expected the override warning, got %q", out)
		}
	})

	t.Run("stays silent when the override names the profile being logged into", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "selected")
		profileAlias = "selected"
		t.Cleanup(func() { profileAlias = "" })

		out := captureStdout(t, printEnvWarning)

		if strings.Contains(out, overrideWarning) {
			t.Fatalf("expected no override warning, got %q", out)
		}
	})

	t.Run("says environment credentials are ignored while the override is set", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "other")
		t.Setenv(session.EnvSpaceliftAPIToken, sampleAPIToken)
		profileAlias = "selected"
		t.Cleanup(func() { profileAlias = "" })

		out := captureStdout(t, printEnvWarning)

		if !strings.Contains(out, "ignored while "+session.EnvSpaceliftProfile+" is set") {
			t.Fatalf("expected the ignored-variables note, got %q", out)
		}
		if strings.Contains(out, "Environment variables take precedence") {
			t.Fatalf("precedence advice is backwards while the override is set, got %q", out)
		}
	})

	t.Run("keeps the precedence advice when no override is set", func(t *testing.T) {
		t.Setenv(session.EnvSpaceliftProfile, "")
		t.Setenv(session.EnvSpaceliftAPIToken, sampleAPIToken)
		profileAlias = "selected"
		t.Cleanup(func() { profileAlias = "" })

		out := captureStdout(t, printEnvWarning)

		if !strings.Contains(out, "Environment variables take precedence") {
			t.Fatalf("expected the precedence advice, got %q", out)
		}
	})
}
