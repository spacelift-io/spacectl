package registry

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
)

func TestRegistryHosts(t *testing.T) {
	for name, tc := range map[string]struct {
		endpoint string
		want     []string
	}{
		"SaaS US":             {"https://acme.app.spacelift.io/graphql", []string{"app.spacelift.io", "spacelift.io"}},
		"SaaS EU":             {"https://acme.app.eu.spacelift.io/graphql", []string{"app.eu.spacelift.io", "eu.spacelift.io"}},
		"spacelift.dev":       {"https://acme.app.spacelift.dev/graphql", []string{"app.spacelift.dev", "spacelift.dev"}},
		"uppercase and slash": {"https://ACME.APP.spacelift.io/", []string{"app.spacelift.io", "spacelift.io"}},
		"self-hosted":         {"https://spacelift.corp.example.com/graphql", []string{"spacelift.corp.example.com"}},
		"self-hosted port":    {"https://spacelift.corp.example.com:8443/graphql", []string{"spacelift.corp.example.com"}},
		"app as first label":  {"https://app.corp.example.com/graphql", []string{"app.corp.example.com"}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := registryHosts(tc.endpoint)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("no host", func(t *testing.T) {
		if _, err := registryHosts("/graphql"); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestTFTokenEnvName(t *testing.T) {
	for host, want := range map[string]string{
		"app.spacelift.io":      "TF_TOKEN_app_spacelift_io",
		"my-registry.corp.com":  "TF_TOKEN_my__registry_corp_com",
		"App.Spacelift.IO":      "TF_TOKEN_app_spacelift_io",
		"app.eu.spacelift.io":   "TF_TOKEN_app_eu_spacelift_io",
		"spacelift.corp-x.com":  "TF_TOKEN_spacelift_corp__x_com",
		"single-label-hostname": "TF_TOKEN_single__label__hostname",
	} {
		if got := tfTokenEnvName(host); got != want {
			t.Errorf("tfTokenEnvName(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestRender(t *testing.T) {
	hosts := []string{"app.spacelift.io", "spacelift.io"}

	for format, want := range map[string]string{
		formatEnv: "TF_TOKEN_app_spacelift_io=tok\nTF_TOKEN_spacelift_io=tok\n",
		formatTerraformRC: "credentials \"app.spacelift.io\" {\n  token = \"tok\"\n}\n" +
			"credentials \"spacelift.io\" {\n  token = \"tok\"\n}\n",
	} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			if err := render(&out, format, hosts, "tok"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.String() != want {
				t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
			}
		})
	}
}

func TestCredentialsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		if _, err := rw.Write([]byte(`{"data":{"apiKeyUser":{"jwt":"SessionJWT","validUntil":4102444800}}}`)); err != nil {
			t.Errorf("could not write mock response: %v", err)
		}
	}))
	defer server.Close()

	// No profile exists in the empty config dir, so the command has to use the
	// environment, as it would in CI.
	t.Setenv(session.EnvSpaceliftConfigDirectory, t.TempDir())
	t.Setenv(session.EnvSpaceliftProfile, "")
	t.Setenv(session.EnvSpaceliftAPIToken, "")
	t.Setenv(session.EnvSpaceliftAPIKeyEndpoint, server.URL)
	t.Setenv(session.EnvSpaceliftAPIKeyID, "key-id")
	t.Setenv(session.EnvSpaceliftAPIKeySecret, "secret")

	run := func(t *testing.T, args ...string) (string, error) {
		t.Helper()

		var out bytes.Buffer
		app := &cli.Command{Name: "spacectl", Writer: &out, Commands: []*cli.Command{Command()}}
		err := app.Run(context.Background(), append([]string{"spacectl", "registry", "credentials"}, args...))
		return out.String(), err
	}

	t.Run("derives the host from the endpoint", func(t *testing.T) {
		out, err := run(t)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := "TF_TOKEN_127_0_0_1=SessionJWT\n"; out != want {
			t.Errorf("got %q, want %q", out, want)
		}
	})

	t.Run("uses the host override", func(t *testing.T) {
		out, err := run(t, "--host", "app.spacelift.io", "--host", "spacelift.io", "--format", formatTerraformRC)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, want := range []string{`credentials "app.spacelift.io"`, `credentials "spacelift.io"`, `token = "SessionJWT"`} {
			if !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		}
	})

	t.Run("rejects an unknown format", func(t *testing.T) {
		out, err := run(t, "--format", "yaml")
		if err == nil || !strings.Contains(err.Error(), `unsupported format "yaml"`) {
			t.Fatalf("expected an unsupported format error, got %v", err)
		}
		if out != "" {
			t.Errorf("printed %q before failing", out)
		}
	})
}
