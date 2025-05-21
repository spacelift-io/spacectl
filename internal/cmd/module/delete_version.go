package module

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func deleteVersion(ctx context.Context, cliCmd *cli.Command) error {
	moduleID := cliCmd.String(flagModuleID.Name)
	versionID := cliCmd.String(flagVersionID.Name)

	var mutation struct {
		DeleteModuleVersion struct {
			Number string `graphql:"number"`
		} `graphql:"versionDelete(id: $id, module: $module)"`
	}

	variables := map[string]interface{}{
		"id":     graphql.ID(versionID),
		"module": graphql.ID(moduleID),
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Module version %q has been deleted\n", mutation.DeleteModuleVersion.Number)
	return nil
}
