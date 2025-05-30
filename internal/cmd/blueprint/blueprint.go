package blueprint

import (
	"context"
	"fmt"
	"math"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// Command returns the blueprint command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:   "blueprint",
		Usage:  "Manage Spacelift blueprints",
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
				Usage: "List the blueprints you have access to",
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
							Action: listBlueprints(),
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
				Usage: "Shows detailed information about a specific blueprint",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagRequiredBlueprintID,
								cmd.FlagOutputFormat,
								cmd.FlagNoColor,
							},
							Action:    (&showCommand{}).show,
							Before:    cmd.PerformAllBefore(cmd.HandleNoColor, authenticated.Ensure),
							ArgsUsage: cmd.EmptyArgsUsage,
						},
					},
				},
			},
			{
				Name:  "deploy",
				Usage: "Deploy a stack from the blueprint",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							Flags: []cli.Flag{
								flagRequiredBlueprintID,
								cmd.FlagNoColor,
							},
							Action:    (&deployCommand{}).deploy,
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
