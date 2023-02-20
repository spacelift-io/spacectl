package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runConfirm() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		stackID := cliCtx.String(flagStackID.Name)

		var mutation struct {
			RunConfirm struct {
				ID string `graphql:"id"`
			} `graphql:"runConfirm(stack: $stack, run: $run)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(cliCtx.String(flagRequiredRun.Name)),
		}

		ctx := context.Background()

		var requestOpts []graphql.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
			return err
		}

		fmt.Println("You have successfully confirmed a deployment")

		fmt.Println("The live run can be visited at", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunConfirm.ID,
		))

		if !cliCtx.Bool(flagTail.Name) {
			return nil
		}

		terminal, err := runLogs(ctx, stackID, mutation.RunConfirm.ID)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
