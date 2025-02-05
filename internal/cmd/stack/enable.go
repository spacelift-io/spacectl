package stack

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type (
	stackEnableMutation struct {
		Stack struct {
			ID       string `graphql:"id"`
			Disabled bool   `graphql:"isDisabled"`
		} `graphql:"stackEnable(id: $stack)"`
	}
	stackDisableMutation struct {
		Stack struct {
			ID       string `graphql:"id"`
			Disabled bool   `graphql:"isDisabled"`
		} `graphql:"stackDisable(id: $stack)"`
	}
)

func enable(cliCtx *cli.Context) error {
	return enableDisable[stackEnableMutation](cliCtx)
}

func disable(cliCtx *cli.Context) error {
	return enableDisable[stackDisableMutation](cliCtx)
}

func enableDisable[T any](cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	if nArgs := cliCtx.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation T
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	return authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)
}
