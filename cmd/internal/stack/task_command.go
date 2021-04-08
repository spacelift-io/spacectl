package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/cmd/internal/authenticated"
)

func taskCommand(cliCtx *cli.Context) error {
	var mutation struct {
		TaskCreate struct {
			ID string `grapqhl:"id"`
		} `graphql:"taskCreate(stack: $stack, command: $command, skipInitialization: $noinit)"`
	}

	variables := map[string]interface{}{
		"stack":   graphql.ID(stackID),
		"command": graphql.String(strings.Join(cliCtx.Args().Slice(), " ")),
		"noinit":  graphql.NewBoolean(graphql.Boolean(cliCtx.Bool(flagNoInit.Name))),
	}

	ctx := context.Background()

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	fmt.Println("You have successfully started a task")

	fmt.Println("The live task can be visited at", authenticated.Client.URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.TaskCreate.ID,
	))

	if !cliCtx.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogs(ctx, stackID, mutation.TaskCreate.ID)
	if err != nil {
		return err
	}

	return terminal.Error()
}
