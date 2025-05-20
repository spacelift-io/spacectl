package provider

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func deleteVersion() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) (err error) {
		versionID := cliCmd.String(flagRequiredVersionID.Name)

		var deleteMutation struct {
			DeleteVersion *internal.Version `graphql:"terraformProviderVersionDelete(version: $version)"`
		}

		variables := map[string]any{"version": graphql.ID(versionID)}

		if err := authenticated.Client.Mutate(ctx, &deleteMutation, variables); err != nil {
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
