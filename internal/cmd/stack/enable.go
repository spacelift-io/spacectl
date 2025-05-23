package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
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

func enable(ctx context.Context, cliCmd *cli.Command) error {
	return enableDisable[stackEnableMutation](ctx, cliCmd)
}

func disable(ctx context.Context, cliCmd *cli.Command) error {
	return enableDisable[stackDisableMutation](ctx, cliCmd)
}

func enableDisable[T any](ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}

	if nArgs := cliCmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation T
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	return authenticated.Client.Mutate(ctx, &mutation, variables)
}
