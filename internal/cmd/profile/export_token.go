package profile

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
)

func exportTokenCommand() *cli.Command {
	return &cli.Command{
		Name: "export-token",
		Usage: "Prints the current token to stdout. In order not to leak, " +
			"we suggest piping it to your OS pastebin",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(ctx context.Context, _ *cli.Command) error {
			sess, err := resolveSession(ctx, manager, http.DefaultClient, os.LookupEnv)
			if err != nil {
				return fmt.Errorf("could not get session: %w", err)
			}

			token, err := sess.BearerToken(ctx)
			if err != nil {
				return fmt.Errorf("could not get bearer token: %w", err)
			}

			if _, err = fmt.Print(token); err != nil {
				return fmt.Errorf("could not print token: %w", err)
			}

			return nil
		},
	}
}

// resolveSession returns the session to export a token from. It prefers the
// currently selected profile, falling back to credentials from the environment
// when none is set (e.g. in CI, where SPACELIFT_API_KEY_ID and
// SPACELIFT_API_KEY_SECRET - the latter possibly an OIDC token - are provided
// directly). The fallback lets `export-token` exchange those for a session token
// without anyone having to hand-roll the apiKeyUser GraphQL mutation.
func resolveSession(ctx context.Context, m *session.ProfileManager, httpClient *http.Client, lookup func(string) (string, bool)) (session.Session, error) {
	if m != nil {
		// Naming a profile through SPACELIFT_PROFILE is an explicit choice, so a bad alias is
		// an error rather than a reason to fall back and print a token for a different account.
		current, err := m.CurrentValidated()
		if err != nil {
			return nil, err
		}

		if current != nil {
			return current.Credentials.Session(ctx, httpClient)
		}
	}

	return session.FromEnvironment(ctx, httpClient)(lookup)
}
