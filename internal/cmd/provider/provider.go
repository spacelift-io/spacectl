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
					gpgKeyFingerprint,
				},
				Action:    createVersion(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category:  "Version management",
				Name:      "delete-version",
				Usage:     "Delete a draft provider version",
				Flags:     []cli.Flag{flagRequiredVersionID},
				Action:    deleteVersion(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Version management",
				Name:     "list-versions",
				Usage:    "List all versions of a provider",
				Flags: []cli.Flag{
					flagProviderType,
					cmd.FlagOutputFormat,
				},
				Action:    listVersions(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category:  "Version management",
				Name:      "publish-version",
				Usage:     "Publish a draft provider version",
				Flags:     []cli.Flag{flagRequiredVersionID},
				Action:    publishVersion(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category:  "Version management",
				Name:      "revoke-version",
				Usage:     "Revoke a published provider version",
				Flags:     []cli.Flag{flagRequiredVersionID},
				Action:    revokeVersion(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}
