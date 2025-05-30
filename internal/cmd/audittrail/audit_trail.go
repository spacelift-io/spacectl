package audittrail

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the audit-trail command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:   "audit-trail",
		Usage:  "Manage Spacelift audit trail entries",
		Before: authenticated.AttemptAutoLogin,
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "list",
				Usage: "List the audit trail entries you have access to",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
								cmd.FlagLimit,
								cmd.FlagSearch,
							},
							Action: listAuditTrails(),
							Before: cmd.PerformAllBefore(
								cmd.HandleNoColor,
								authenticated.Ensure,
							),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}
