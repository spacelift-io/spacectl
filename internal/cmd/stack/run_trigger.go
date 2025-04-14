package stack

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runTrigger(spaceliftType, humanType string) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		stackID, err := getStackID(cliCtx)
		if err != nil {
			return err
		}

		var runtimeConfigInput *RuntimeConfigInput
		if cliCtx.IsSet(flagRuntimeConfig.Name) {
			runtimeConfigFilePath := cliCtx.String(flagRuntimeConfig.Name)

			if _, err = os.Stat(runtimeConfigFilePath); err != nil {
				return fmt.Errorf("runtime config file does not exist: %v", err)
			}

			data, err := os.ReadFile(runtimeConfigFilePath)
			if err != nil {
				return fmt.Errorf("failed to read runtime config file: %v", err)
			}

			yaml := string(data)

			runtimeConfigInput = &RuntimeConfigInput{
				Yaml: &yaml,
			}
		}

		var mutation struct {
			RunTrigger struct {
				ID string `graphql:"id"`
			} `graphql:"runTrigger(stack: $stack, commitSha: $sha, runType: $type, runtimeConfig: $runtimeConfig)"`
		}

		variables := map[string]interface{}{
			"stack":         graphql.ID(stackID),
			"sha":           (*graphql.String)(nil),
			"type":          structs.NewRunType(spaceliftType),
			"runtimeConfig": runtimeConfigInput,
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

		if !cliCtx.Bool(flagTail.Name) && !cliCtx.Bool(flagAutoConfirm.Name) {
			return nil
		}

		actionFn := func(state structs.RunState, stackID, runID string) error {
			if state != "UNCONFIRMED" {
				return nil
			}

			if !cliCtx.Bool(flagAutoConfirm.Name) {
				return nil
			}

			var mutation struct {
				RunConfirm struct {
					ID string `graphql:"id"`
				} `graphql:"runConfirm(stack: $stack, run: $run)"`
			}

			variables := map[string]interface{}{
				"stack": graphql.ID(stackID),
				"run":   graphql.ID(runID),
			}

			if err := authenticated.Client.Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
				return err
			}

			fmt.Println("Deployment was automatically confirmed because of --auto-confirm flag")
			return nil
		}

		terminal, err := runLogsWithAction(ctx, stackID, mutation.RunTrigger.ID, actionFn)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}
