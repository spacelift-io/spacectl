package provider

import (
	"fmt"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func listGPGKeys() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		var query struct {
			GPGKeys internal.GPGKeys `graphql:"gpgKeys"`
		}

		if err := authenticated.Client.Query(cliCtx.Context, &query, nil); err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatJSON:
			return cmd.OutputJSON(query)
		case cmd.OutputFormatTable:
			rows := [][]string{query.GPGKeys.Headers()}

			for _, key := range query.GPGKeys {
				rows = append(rows, key.Row())
			}

			return cmd.OutputTable(rows, true)
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}
	}
}
