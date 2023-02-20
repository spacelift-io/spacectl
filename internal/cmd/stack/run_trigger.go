package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runTrigger(spaceliftType, humanType string) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		stackID := cliCtx.String(flagStackID.Name)

		var mutation struct {
			RunTrigger struct {
				ID string `graphql:"id"`
			} `graphql:"runTrigger(stack: $stack, commitSha: $sha, runType: $type)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"sha":   (*graphql.String)(nil),
			"type":  structs.NewRunType(spaceliftType),
		}

		if cliCtx.IsSet(flagCommitSHA.Name) {
			variables["sha"] = graphql.NewString(graphql.String(cliCtx.String(flagCommitSHA.Name)))
		}

		ctx := context.Background()

		var requestOpts []graphql.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
			return err
		}

		fmt.Println("You have successfully created a", humanType)

		fmt.Println("The live run can be visited at", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunTrigger.ID,
		))

		if !cliCtx.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := runLogs(ctx, stackID, mutation.RunTrigger.ID)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
