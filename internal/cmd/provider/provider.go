package provider

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the provider command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "provider",
		Usage: "Manage a Terraform provider",
		Subcommands: []*cli.Command{
			{
				Category: "Version management",
				Name:     "create-version",
				Usage:    "Create a new provider version, designed to be called from a CI/CD pipeline",
				Flags: []cli.Flag{
					flagProviderType,
					flagProviderVersionProtocols,
					flagGoReleaserDir,
					flagSigningKeyID,
				},
				Action:    createVersion(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}
