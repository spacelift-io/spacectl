package provider

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
	"github.com/urfave/cli/v2"
)

func publishVersion() cli.ActionFunc {
	return func(cliCtx *cli.Context) (err error) {
		versionID := cliCtx.String(flagRequiredVersionID.Name)

		var publishMutation struct {
			PublishVersion internal.Version `graphql:"terraformProviderVersionPublish(version: $version)"`
		}

		variables := map[string]any{"version": graphql.ID(versionID)}

		if err := authenticated.Client.Mutate(cliCtx.Context, &publishMutation, variables); err != nil {
			return fmt.Errorf("could not publish Terraform provider version: %w", err)
		}

		_, err = fmt.Printf("Terraform provider version %s published", publishMutation.PublishVersion.Number)
		return
	}
}
