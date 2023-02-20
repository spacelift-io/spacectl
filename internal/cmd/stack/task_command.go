package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func taskCommand(cliCtx *cli.Context) error {
	stackID := cliCtx.String(flagStackID.Name)

	var mutation struct {
		TaskCreate struct {
			ID string `graphql:"id"`
		} `graphql:"taskCreate(stack: $stack, command: $command, skipInitialization: $noinit)"`
	}

	variables := map[string]interface{}{
		"stack":   graphql.ID(stackID),
		"command": graphql.String(strings.Join(cliCtx.Args().Slice(), " ")),
		"noinit":  graphql.NewBoolean(graphql.Boolean(cliCtx.Bool(flagNoInit.Name))),
	}

	ctx := context.Background()

	var requestOpts []graphql.RequestOption
	if cliCtx.IsSet(flagRunMetadata.Name) {
		requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
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
