package provider

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command encapsulates the provider command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "provider",
		Usage: "Manage a Terraform provider",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Category: "GPG key management",
				Name:     "add-gpg-key",
				Usage:    "Adds a new GPG key for signing provider releases",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagKeyEmail,
								flagKeyGenerate,
								flagKeyImport,
								flagKeyName,
								flagKeyPath,
							},
							Action:    addGPGKey(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "GPG key management",
				Name:     "list-gpg-keys",
				Usage:    "List all GPG keys registered in the account",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags:     []cli.Flag{cmd.FlagOutputFormat},
							Action:    listGPGKeys(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "GPG key management",
				Name:     "revoke-gpg-key",
				Usage:    "Revoke a GPG key",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags:     []cli.Flag{flagKeyID},
							Action:    revokeGPGKey(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Version management",
				Name:     "create-version",
				Usage:    "Create a new provider version, designed to be called from a CI/CD pipeline",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagProviderType,
								flagProviderVersionProtocols,
								flagGoReleaserDir,
								flagGPGKeyID,
							},
							Action:    createVersion(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Version management",
				Name:     "delete-version",
				Usage:    "Delete a draft provider version",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags:     []cli.Flag{flagRequiredVersionID},
							Action:    deleteVersion(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Version management",
				Name:     "list-versions",
				Usage:    "List all versions of a provider",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagProviderType,
								cmd.FlagOutputFormat,
							},
							Action:    listVersions(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Version management",
				Name:     "publish-version",
				Usage:    "Publish a draft provider version",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags:     []cli.Flag{flagRequiredVersionID},
							Action:    publishVersion(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Category: "Version management",
				Name:     "revoke-version",
				Usage:    "Revoke a published provider version",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags:     []cli.Flag{flagRequiredVersionID},
							Action:    revokeVersion(),
							Before:    authenticated.Ensure,
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}
