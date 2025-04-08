package provider

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func listGPGKeys() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		outputFormat, err := internalCmd.GetOutputFormat(cmd)
		if err != nil {
			return err
		}

		var query struct {
			GPGKeys internal.GPGKeys `graphql:"gpgKeys"`
		}

		if err := authenticated.Client.Query(ctx, &query, nil); err != nil {
			return err
		}

		switch outputFormat {
		case internalCmd.OutputFormatJSON:
			return internalCmd.OutputJSON(query)
		case internalCmd.OutputFormatTable:
			rows := [][]string{query.GPGKeys.Headers()}

			for _, key := range query.GPGKeys {
				rows = append(rows, key.Row())
			}

			return internalCmd.OutputTable(rows, true)
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}
	}
}
