package whoami

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the logged-in user's information.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "whoami",
		Usage: "Print out logged-in user's information",
		Action: func(cliCtx *cli.Context) error {
			manager, err := session.UserProfileManager()
			if err != nil {
				return fmt.Errorf("could not access profile manager: %w", err)
			}
			var endpoint string
			if p := manager.Current(); p != nil {
				endpoint = p.Credentials.Endpoint
			}

			var query struct {
				Viewer *struct {
					ID   string `graphql:"id" json:"id"`
					Name string `graphql:"name" json:"name"`
				}
			}
			if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{}); err != nil {
				return errors.Wrap(err, "failed to query user information")
			}
			if query.Viewer == nil {
				return errors.New("failed to query user information: unauthorized")
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "    ")
			return encoder.Encode(struct {
				ID       string `json:"id,omitempty"`
				Name     string `json:"name,omitempty"`
				Endpoint string `json:"endpoint,omitempty"`
			}{
				ID:       query.Viewer.ID,
				Name:     query.Viewer.Name,
				Endpoint: endpoint,
			})
		},
		Before:    authenticated.Ensure,
		ArgsUsage: cmd.EmptyArgsUsage,
	}
}
