package module

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func deleteVersion(cliCtx *cli.Context) error {
	moduleID := cliCtx.String(flagModuleID.Name)
	versionID := cliCtx.String(flagVersionID.Name)

	var mutation struct {
		DeleteModuleVersion struct {
			Number string `graphql:"number"`
		} `graphql:"versionDelete(id: $id, module: $module)"`
	}

	variables := map[string]interface{}{
		"id":     graphql.ID(versionID),
		"module": graphql.ID(moduleID),
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Module version %q has been deleted\n", mutation.DeleteModuleVersion.Number)
	return nil
}
