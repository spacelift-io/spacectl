package module

import (
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
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
		},
	}
}
