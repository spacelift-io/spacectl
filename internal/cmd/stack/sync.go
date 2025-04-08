package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func syncCommit(ctx context.Context, cmd *cli.Command) error {
	stackID, err := getStackID(ctx, cmd)
	if err != nil {
		return err
	}

	if nArgs := cmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation struct {
		Stack struct {
			ID string `graphql:"id"`
		} `graphql:"stackSyncCommit(id: $stack)"`
	}
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	return authenticated.Client.Mutate(ctx, &mutation, variables)
}
