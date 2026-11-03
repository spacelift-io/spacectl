package stack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
)

func TestSetVarDescription(t *testing.T) {
	// setVarAPI fakes the GraphQL API. existing is the stored description of the
	// variable (nil means the variable has none), and the config sent with the
	// stackConfigAdd mutation is recorded in sent.
	type setVarAPI struct {
		existing     *string
		variableSeen bool
		sent         map[string]any
	}

	run := func(t *testing.T, api *setVarAPI, args ...string) {
		t.Helper()

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			var request struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("could not decode request: %v", err)
			}

			var response any
			switch {
			case strings.Contains(request.Query, "apiKeyUser"):
				response = map[string]any{"apiKeyUser": map[string]any{"jwt": "SessionJWT", "validUntil": 4102444800}}
			case strings.Contains(request.Query, "stackConfigAdd"):
				api.sent, _ = request.Variables["config"].(map[string]any)
				response = map[string]any{"stackConfigAdd": map[string]any{"id": "FOO", "writeOnly": false, "value": "bar"}}
			case strings.Contains(request.Query, "configElement"):
				var element any
				if api.variableSeen {
					element = map[string]any{"description": api.existing}
				}
				response = map[string]any{"stack": map[string]any{"configElement": element}}
			default:
				response = map[string]any{"stack": map[string]any{"id": "my-stack", "name": "My stack"}}
			}

			if err := json.NewEncoder(rw).Encode(map[string]any{"data": response}); err != nil {
				t.Errorf("could not write mock response: %v", err)
			}
		}))
		defer server.Close()

		t.Setenv(session.EnvSpaceliftConfigDirectory, t.TempDir())
		t.Setenv(session.EnvSpaceliftProfile, "")
		t.Setenv(session.EnvSpaceliftAPIToken, "")
		t.Setenv(session.EnvSpaceliftAPIKeyEndpoint, server.URL)
		t.Setenv(session.EnvSpaceliftAPIKeyID, "key-id")
		t.Setenv(session.EnvSpaceliftAPIKeySecret, "secret")

		app := &cli.Command{
			Name:     "spacectl",
			Commands: cmd.ResolveCommands(cmd.SpaceliftInstanceVersion{InstanceType: cmd.SpaceliftInstanceTypeSaaS}, []cmd.Command{Command()}),
		}
		args = append([]string{"spacectl", "stack", "environment", "setvar", "--id", "my-stack"}, args...)
		if err := app.Run(context.Background(), args); err != nil {
			t.Fatalf("setvar failed: %v", err)
		}
		if api.sent == nil {
			t.Fatal("stackConfigAdd was not called")
		}
	}

	existing := "Used by the deploy script"

	t.Run("keeps the description of an existing variable", func(t *testing.T) {
		api := &setVarAPI{existing: &existing, variableSeen: true}
		run(t, api, "FOO", "bar")
		if got := api.sent["description"]; got != existing {
			t.Errorf("sent description %v, want %q", got, existing)
		}
	})

	t.Run("sends no description for a new variable", func(t *testing.T) {
		api := &setVarAPI{}
		run(t, api, "FOO", "bar")
		if _, ok := api.sent["description"]; ok {
			t.Errorf("sent description %v for a new variable", api.sent["description"])
		}
	})

	// Flags are package-level values that keep their state between app.Run
	// calls, so the case that sets --description runs last.
	t.Run("uses the description flag over the existing one", func(t *testing.T) {
		api := &setVarAPI{existing: &existing, variableSeen: true}
		run(t, api, "--description", "New description", "FOO", "bar")
		if got := api.sent["description"]; got != "New description" {
			t.Errorf("sent description %v, want %q", got, "New description")
		}
	})
}
