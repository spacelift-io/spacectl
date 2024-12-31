package provider

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func listVersions() cli.ActionFunc {
	return func(cliCtx *cli.Context) (err error) {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		var query struct {
			TerraformProvider *struct {
				Versions internal.Versions `graphql:"versions"`
			} `graphql:"terraformProvider(id: $id)"`
		}

		providerType := cliCtx.String(flagProviderType.Name)

		variables := map[string]any{"id": graphql.ID(providerType)}
		if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
			return fmt.Errorf("could not list Terraform provider versions: %w", err)
		}

		if query.TerraformProvider == nil {
			return fmt.Errorf("provider %s not found", providerType)
		}

		versions := query.TerraformProvider.Versions

		switch outputFormat {
		case cmd.OutputFormatJSON:
			return cmd.OutputJSON(map[string]any{"versions": versions})
		case cmd.OutputFormatTable:
			rows := [][]string{versions.Headers()}
			for _, version := range versions {
				rows = append(rows, version.Row())
			}

			return cmd.OutputTable(rows, true)
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}
	}
}
