package provider

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

func publishVersion() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) (err error) {
		versionID := cliCmd.String(flagRequiredVersionID.Name)

		var publishMutation struct {
			PublishVersion internal.Version `graphql:"terraformProviderVersionPublish(version: $version)"`
		}

		variables := map[string]any{"version": graphql.ID(versionID)}

		if err := authenticated.Client.Mutate(ctx, &publishMutation, variables); err != nil {
			return fmt.Errorf("could not publish Terraform provider version: %w", err)
		}

		_, err = fmt.Printf("Terraform provider version %s published", publishMutation.PublishVersion.Number)
		return
	}
}
