package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func taskCommand(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd, nil)
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
		"command": graphql.String(strings.Join(cliCmd.Args().Slice(), " ")),
		"noinit":  graphql.NewBoolean(graphql.Boolean(cliCmd.Bool(flagNoInit.Name))),
	}

	var requestOpts []graphql.RequestOption
	if cliCmd.IsSet(flagRunMetadata.Name) {
		requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCmd.String(flagRunMetadata.Name)))
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
		return err
	}

	fmt.Println("You have successfully started a task")

	fmt.Println("The live task can be visited at", authenticated.Client().URL(
		"/stack/%s/run/%s",
		stackID,
		mutation.TaskCreate.ID,
	))

	if !cliCmd.Bool(flagTail.Name) {
		return nil
	}

	terminal, err := logs.NewExplorer(stackID, mutation.TaskCreate.ID).RunFilteredLogs(ctx)
	if err != nil {
		return err
	}

	return terminal.Error()
}
