package module

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the module command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "module",
		Usage: "Manage a Spacelift module",
		Subcommands: []*cli.Command{
			{
				Category: "Module management",
				Name:     "create-version",
				Usage:    "Create a new version of a module",
				Flags: []cli.Flag{
					flagModuleID,
					flagCommitSHA,
					flagVersion,
				},
				Action:    createVersion,
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Module management",
				Name:     "local-preview",
				Usage:    "Start a preview (proposed version) based on the current project. Respects .gitignore and .terraformignore.",
				Flags: []cli.Flag{
					flagModuleID,
					flagNoFindRepositoryRoot,
					flagNoUpload,
					flagRunMetadata,
					flagDisregardGitignore,
					flagTests,
				},
				Action:    localPreview(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Module management",
				Name:     "list",
				Usage:    "List all modules available and their current version",
				Flags: []cli.Flag{
					cmd.FlagOutputFormat,
				},
				Action:    listModules(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
			{
				Category: "Module management",
				Name:     "list-versions",
				Usage:    "List 20 latest non failed versions for a module",
				Flags: []cli.Flag{
					flagModuleID,
					cmd.FlagOutputFormat,
				},
				Action:    listVersions(),
				Before:    authenticated.Ensure,
				ArgsUsage: cmd.EmptyArgsUsage,
			},
		},
	}
}
