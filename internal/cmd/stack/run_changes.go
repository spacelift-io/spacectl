package stack

import (
	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func runChanges(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	run := cliCtx.String(flagRequiredRun.Name)

	result, err := getRunChanges(cliCtx, stackID, run)
	if err != nil {
		return err
	}

	return cmd.OutputJSON(result)
}

func getRunChanges(cliCtx *cli.Context, stackID, runID string) ([]runChangesData, error) {
	var query struct {
		Stack struct {
			Run struct {
				ChangesV3 []runChangesData `graphql:"changesV3(input: {})"`
			} `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $stack)"`
	}

	variables := map[string]any{
		"stack": graphql.ID(stackID),
		"run":   graphql.ID(runID),
	}
	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to query one stack")
	}

	return query.Stack.Run.ChangesV3, nil
}

type runChangesData struct {
	Resources []runChangesResource `graphql:"resources"`
}

type runChangesResource struct {
	Address         string             `graphql:"address"`
	PreviousAddress string             `graphql:"previousAddress"`
	Metadata        runChangesMetadata `graphql:"metadata"`
}

type runChangesMetadata struct {
	Type string `graphql:"type"`
}
