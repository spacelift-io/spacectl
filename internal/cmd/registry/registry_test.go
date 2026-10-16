package registry

import (
	"bytes"
	"context"
	"io"
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
		endpoint   string
		selfHosted bool
		want       []string
	}{
		"SaaS":                       {"https://acme.app.spacelift.io/graphql", false, []string{"app.spacelift.io", "spacelift.io"}},
		"SaaS spacelift.dev":         {"https://acme.app.spacelift.dev/graphql", false, []string{"app.spacelift.dev", "spacelift.dev"}},
		"SaaS uppercase and slash":   {"https://ACME.APP.spacelift.io/", false, []string{"app.spacelift.io", "spacelift.io"}},
		"SaaS default port":          {"https://acme.app.spacelift.io:443/graphql", false, []string{"app.spacelift.io", "spacelift.io"}},
		"SaaS domain without app":    {"https://acme.spacelift.example.com/graphql", false, []string{"spacelift.example.com"}},
		"self-hosted":                {"https://spacelift.corp.example.com/graphql", true, []string{"spacelift.corp.example.com"}},
		"self-hosted with app label": {"https://spacelift.app.corp.example.com/graphql", true, []string{"spacelift.app.corp.example.com"}},
		"self-hosted port":           {"https://spacelift.corp.example.com:8443/graphql", true, []string{"spacelift.corp.example.com:8443"}},
		"self-hosted default port":   {"https://spacelift.corp.example.com:443/graphql", true, []string{"spacelift.corp.example.com"}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := registryHosts(tc.endpoint, tc.selfHosted)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}

	for name, endpoint := range map[string]string{
		"no host":                "/graphql",
		"SaaS without subdomain": "https://spacelift.io/graphql",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := registryHosts(endpoint, false); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
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

func TestRenderRejectsPortInEnvFormat(t *testing.T) {
	var out bytes.Buffer
	err := render(&out, formatEnv, []string{"spacelift.io", "spacelift.corp.example.com:8443"}, "tok")
	if err == nil || !strings.Contains(err.Error(), "--format terraformrc") {
		t.Fatalf("expected an error pointing at --format terraformrc, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("printed %q before failing", out.String())
	}
}

func TestCredentialsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("could not read request: %v", err)
		}

		response := `{"data":{"apiKeyUser":{"jwt":"SessionJWT","validUntil":4102444800}}}`
		if strings.Contains(string(body), "debugInfo") {
			response = `{"data":{"debugInfo":{"selfHostedVersion":"3.0.0"}}}`
		}

		if _, err := rw.Write([]byte(response)); err != nil {
			t.Errorf("could not write mock response: %v", err)
		}
	}))
	defer server.Close()

	// Use an empty config directory to exercise CI authentication without picking
	// up a profile from the developer's machine.
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

	t.Run("derives the self-hosted host and port", func(t *testing.T) {
		out, err := run(t, "--format", formatTerraformRC)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		host := strings.TrimPrefix(server.URL, "http://")
		if want := "credentials \"" + host + "\" {\n  token = \"SessionJWT\"\n}\n"; out != want {
			t.Errorf("got %q, want %q", out, want)
		}
	})

	t.Run("refuses env format for a host with a port", func(t *testing.T) {
		out, err := run(t)
		if err == nil || !strings.Contains(err.Error(), "has a port") {
			t.Fatalf("expected a port error, got %v", err)
		}
		if out != "" {
			t.Errorf("printed %q before failing", out)
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
