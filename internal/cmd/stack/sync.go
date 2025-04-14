package stack

import (
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func syncCommit(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	if nArgs := cliCtx.NArg(); nArgs != 0 {
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

	return authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)
}
