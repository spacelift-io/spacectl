package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

var flagDestroyResources = &cli.BoolFlag{
	Name:  "destroy-resources",
	Usage: "Indicate whether to destroy stack resources during deletion",
	Value: false,
}

func deleteStack() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		stackID, err := getStackID(cliCtx)
		if err != nil {
			return err
		}

		destroyResources := cliCtx.Bool(flagDestroyResources.Name)

		fmt.Printf("Deleting stack %s\n", stackID)
		if destroyResources {
			fmt.Println("Resources managed by this stack will also be destroyed.")
		}

		var mutation struct {
			StackDelete struct {
				ID string `graphql:"id"`
			} `graphql:"stackDelete(id: $id, destroyResources: $destroyResources)"`
		}

		variables := map[string]interface{}{
			"id":               graphql.ID(stackID),
			"destroyResources": graphql.Boolean(destroyResources),
		}

		ctx := context.Background()

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("Stack has been successfully deleted")

		return nil
	}
}
