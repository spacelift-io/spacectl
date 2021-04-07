package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/client/structs"
	"github.com/spacelift-io/spacelift-cli/cmd/internal/authenticated"
)

func runTrigger(spaceliftType, humanType string) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		var mutation struct {
			RunTrigger struct {
				ID string `grapqhl:"id"`
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

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
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

		return runLogs(ctx, stackID, mutation.RunTrigger.ID)
	}
}
