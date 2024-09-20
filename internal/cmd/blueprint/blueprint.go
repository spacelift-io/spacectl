package blueprint

import (
	"fmt"
	"math"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

// Command encapsulates the blueprintNode command subtree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "blueprint",
		Usage: "Manage a Spacelift blueprints",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List the blueprints you have access to",
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
	}
}

func validateLimit(cliCtx *cli.Context) error {
	if cliCtx.IsSet(cmd.FlagLimit.Name) {
		if cliCtx.Uint(cmd.FlagLimit.Name) == 0 {
			return fmt.Errorf("limit must be greater than 0")
		}

		if cliCtx.Uint(cmd.FlagLimit.Name) >= math.MaxInt32 {
			return fmt.Errorf("limit must be less than %d", math.MaxInt32)
		}
	}

	return nil
}

func validateSearch(cliCtx *cli.Context) error {
	if cliCtx.IsSet(cmd.FlagSearch.Name) {
		if cliCtx.String(cmd.FlagSearch.Name) == "" {
			return fmt.Errorf("search must be non-empty")
		}

	}

	return nil
}
