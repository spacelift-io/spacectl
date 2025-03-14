package audittrail

import (
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "audit-trail",
		Usage: "Manage a Spacelift audit trail entries",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the audit trail entries you have access to",
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
	}
}
