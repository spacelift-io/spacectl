package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func taskCommand(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
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
		"command": strings.Join(cliCtx.Args().Slice(), " "),
		"noinit":  internal.Ptr(cliCtx.Bool(flagNoInit.Name)),
	}

	ctx := context.Background()

	var requestOpts []client.RequestOption
	if cliCtx.IsSet(flagRunMetadata.Name) {
		requestOpts = append(requestOpts, client.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
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

	terminal, err := runLogsWithAction(ctx, stackID, mutation.TaskCreate.ID, nil)
	if err != nil {
		return err
	}

	return terminal.Error()
}
