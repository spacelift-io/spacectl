package provider

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func listVersions() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) (err error) {
		outputFormat, err := internalCmd.GetOutputFormat(cmd)
		if err != nil {
			return err
		}

		var query struct {
			TerraformProvider *struct {
				Versions internal.Versions `graphql:"versions"`
			} `graphql:"terraformProvider(id: $id)"`
		}

		providerType := cmd.String(flagProviderType.Name)

		variables := map[string]any{"id": graphql.ID(providerType)}
		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return fmt.Errorf("could not list Terraform provider versions: %w", err)
		}

		if query.TerraformProvider == nil {
			return fmt.Errorf("provider %s not found", providerType)
		}

		versions := query.TerraformProvider.Versions

		switch outputFormat {
		case internalCmd.OutputFormatJSON:
			return internalCmd.OutputJSON(map[string]any{"versions": versions})
		case internalCmd.OutputFormatTable:
			rows := [][]string{versions.Headers()}
			for _, version := range versions {
				rows = append(rows, version.Row())
			}

			return internalCmd.OutputTable(rows, true)
		default:
			return fmt.Errorf("unknown output format: %s", outputFormat)
		}
	}
}
