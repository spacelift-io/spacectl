package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func taskCommand(ctx context.Context, cmd *cli.Command) error {
	stackID, err := getStackID(ctx, cmd)
	if err != nil {
		return err
	}

	var mutation struct {
		TaskCreate struct {
			ID string `graphql:"id"`
		} `graphql:"taskCreate(stack: $stack, command: $command, skipInitialization: $noinit)"`
	}

	variables := map[string]interface{}{
		"stack":   graphql.ID(stackID),
		"command": graphql.String(strings.Join(cmd.Args().Slice(), " ")),
		"noinit":  graphql.NewBoolean(graphql.Boolean(cmd.Bool(flagNoInit.Name))),
	}

	var requestOpts []graphql.RequestOption
	if cmd.IsSet(flagRunMetadata.Name) {
		requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cmd.String(flagRunMetadata.Name)))
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

	if !cmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := runLogsWithAction(ctx, stackID, mutation.TaskCreate.ID, nil)
	if err != nil {
		return err
	}

	return terminal.Error()
}
