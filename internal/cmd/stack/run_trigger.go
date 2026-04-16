package stack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
)

func runTrigger(spaceliftType, humanType string) cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd)
		if err != nil {
			return err
		}

		var runtimeConfigInput *RuntimeConfigInput
		if cliCmd.IsSet(flagRuntimeConfig.Name) {
			runtimeConfigFilePath := cliCmd.String(flagRuntimeConfig.Name)

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

		var forceApply *structs.ForceApplyMode
		if cliCmd.IsSet(flagForceApply.Name) {
			forceApply, err = parseForceApplyMode(cliCmd.String(flagForceApply.Name))
			if err != nil {
				return err
			}
		}

		var mutation struct {
			RunTrigger struct {
				ID string `graphql:"id"`
			} `graphql:"runTrigger(stack: $stack, commitSha: $sha, runType: $type, runtimeConfig: $runtimeConfig, forceApply: $forceApply)"`
		}

		variables := map[string]any{
			"stack":         graphql.ID(stackID),
			"sha":           (*graphql.String)(nil),
			"type":          structs.NewRunType(spaceliftType),
			"runtimeConfig": runtimeConfigInput,
			"forceApply":    forceApply,
		}

		if cliCmd.IsSet(flagCommitSHA.Name) {
			variables["sha"] = new(graphql.String(cliCmd.String(flagCommitSHA.Name)))
		}

		var requestOpts []graphql.RequestOption
		if cliCmd.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCmd.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client().Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
			return err
		}

		fmt.Println("You have successfully created a", humanType)

		fmt.Println("The live run can be visited at", authenticated.Client().URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunTrigger.ID,
		))

		if !cliCmd.Bool(flagTail.Name) && !cliCmd.Bool(flagAutoConfirm.Name) {
			return nil
		}

		actionFn := func(state structs.RunState, stackID, runID string) error {
			if state != "UNCONFIRMED" {
				return nil
			}

			if !cliCmd.Bool(flagAutoConfirm.Name) {
				return nil
			}

			var mutation struct {
				RunConfirm struct {
					ID string `graphql:"id"`
				} `graphql:"runConfirm(stack: $stack, run: $run)"`
			}

			variables := map[string]any{
				"stack": graphql.ID(stackID),
				"run":   graphql.ID(runID),
			}

			if err := authenticated.Client().Mutate(ctx, &mutation, variables, requestOpts...); err != nil {
				return err
			}

			fmt.Println("Deployment was automatically confirmed because of --auto-confirm flag")
			return nil
		}

		terminal, err := logs.NewExplorer(stackID, mutation.RunTrigger.ID, logs.WithActionOnRunState(actionFn)).RunFilteredLogs(ctx)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}

func parseForceApplyMode(s string) (*structs.ForceApplyMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "single":
		return structs.NewForceApplyMode("SINGLE"), nil
	case "cascade":
		return structs.NewForceApplyMode("CASCADE"), nil
	default:
		return nil, fmt.Errorf("invalid --force-apply value %q (use single or cascade)", s)
	}
}
