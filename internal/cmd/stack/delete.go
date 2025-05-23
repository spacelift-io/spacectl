package stack

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var flagDestroyResources = &cli.BoolFlag{
	Name:  "destroy-resources",
	Usage: "Indicate whether to destroy stack resources during deletion",
	Value: false,
}

var flagSkipConfirmation = &cli.BoolFlag{
	Name:  "skip-confirmation",
	Usage: "Whether to skip confirmation prompt before deleting the stack",
	Value: false,
}

func deleteStack() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd)
		if err != nil {
			return err
		}

		if !cliCmd.Bool(flagSkipConfirmation.Name) {
			fmt.Print("Are you sure you want to delete this stack? (y/n): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return err
			}

			if strings.TrimSpace(response) != "y" {
				fmt.Println("Stack deletion aborted.")
				return nil
			}
		}

		destroyResources := cliCmd.Bool(flagDestroyResources.Name)

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

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Println("Stack has been successfully deleted")

		return nil
	}
}
