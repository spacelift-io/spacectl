package template

import (
	"context"
	"fmt"
	"math"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the template command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "template",
		Usage: "Manage Spacelift templates",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "list",
				Usage: "List the templates you have access to",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								cmd.FlagShowLabels,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
								cmd.FlagLimit,
								cmd.FlagSearch,
							},
							Action: listTemplates,
							Before: cmd.PerformAllBefore(
								cmd.HandleNoColor,
								authenticated.Ensure,
								validateLimit,
								validateSearch,
							),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "show",
				Usage: "Shows detailed information about a specific template",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagRequiredTemplateID,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
							},
							Action:    showTemplate,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
		},
	}
}

func validateLimit(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
	if cliCmd.IsSet(cmd.FlagLimit.Name) {
		if cliCmd.Uint(cmd.FlagLimit.Name) == 0 {
			return ctx, fmt.Errorf("limit must be greater than 0")
		}

		if cliCmd.Uint(cmd.FlagLimit.Name) >= math.MaxInt32 {
			return ctx, fmt.Errorf("limit must be less than %d", math.MaxInt32)
		}
	}

	return ctx, nil
}

func validateSearch(ctx context.Context, cliCmd *cli.Command) (context.Context, error) {
	if cliCmd.IsSet(cmd.FlagSearch.Name) {
		if cliCmd.String(cmd.FlagSearch.Name) == "" {
			return ctx, fmt.Errorf("search must be non-empty")
		}
	}

	return ctx, nil
}
