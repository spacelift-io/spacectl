package provider

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func deleteVersion() cli.ActionFunc {
	return func(cliCtx *cli.Context) (err error) {
		versionID := cliCtx.String(flagRequiredVersionID.Name)

		var deleteMutation struct {
			DeleteVersion *internal.Version `graphql:"terraformProviderVersionDelete(version: $version)"`
		}

		variables := map[string]any{"version": graphql.ID(versionID)}

		if err := authenticated.Client.Mutate(cliCtx.Context, &deleteMutation, variables); err != nil {
			return fmt.Errorf("could not delete Terraform provider version: %w", err)
		}

		if deleteMutation.DeleteVersion == nil {
			_, err = fmt.Printf("Terraform provider version %s not found", versionID)
			return
		}

		_, err = fmt.Printf("Terraform provider version %s deleted", deleteMutation.DeleteVersion.Number)
		return
	}
}
