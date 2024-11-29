package provider

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the provider command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "provider",
		Usage: "Manage a Terraform provider",
		Subcommands: []*cli.Command{
			{
				Category: "GPG key management",
				Name:     "add-gpg-key",
				Usage:    "Adds a new GPG key for signing provider releases",
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
			{
				Category:  "GPG key management",
				Name:      "list-gpg-keys",
				Usage:     "List all GPG keys registered in the account",
				Flags:     []cli.Flag{cmd.FlagOutputFormat},
				Action:    listGPGKeys(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category:  "GPG key management",
				Name:      "revoke-gpg-key",
				Usage:     "Revoke a GPG key",
				Flags:     []cli.Flag{flagKeyID},
				Action:    revokeGPGKey(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Version management",
				Name:     "create-version",
				Usage:    "Create a new provider version, designed to be called from a CI/CD pipeline",
				Flags: []cli.Flag{
					flagProviderType,
					flagProviderVersionProtocols,
					flagGoReleaserDir,
					flagGPGKeyID,
					flagUseRegisterPlatformV2,
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
