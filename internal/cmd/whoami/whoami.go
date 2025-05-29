package whoami

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the logged-in user's information.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "whoami",
		Usage: "Print out logged-in user's information",
		Action: func(ctx context.Context, _ *cli.Command) error {
			manager, err := session.UserProfileManager()
			if err != nil {
				return fmt.Errorf("could not access profile manager: %w", err)
			}
			var endpoint string
			if p := manager.Current(); p != nil {
				endpoint = p.Credentials.Endpoint
			}

			query, err := authenticated.CurrentViewer(ctx)
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "    ")
			return encoder.Encode(struct {
				ID       string `json:"id,omitempty"`
				Name     string `json:"name,omitempty"`
				Endpoint string `json:"endpoint,omitempty"`
			}{
				ID:       query.ID,
				Name:     query.Name,
				Endpoint: endpoint,
			})
		},
		Before:    authenticated.Ensure,
		ArgsUsage: cmd.EmptyArgsUsage,
	}
}
