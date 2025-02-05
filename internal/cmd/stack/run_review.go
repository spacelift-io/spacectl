package stack

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/client/enums"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type runReviewMutation struct {
	Review struct {
		ID string `graphql:"id"`
	} `graphql:"runReview(stack: $stack, run: $run, decision: $decision, note: $note)"`
}

var flagRunReviewNote = &cli.StringFlag{
	Name:     "note",
	Usage:    "Description of why the review decision was made.",
	Required: false,
}

func runApprove(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	runID := cliCtx.String(flagRequiredRun.Name)
	note := cliCtx.String(flagRunReviewNote.Name)

	if nArgs := cliCtx.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	return addRunReview(cliCtx.Context, stackID, runID, note, enums.RunReviewDecisionApprove)
}

func runReject(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	runID := cliCtx.String(flagRequiredRun.Name)
	note := cliCtx.String(flagRunReviewNote.Name)

	if nArgs := cliCtx.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	return addRunReview(cliCtx.Context, stackID, runID, note, enums.RunReviewDecisionReject)
}

func addRunReview(ctx context.Context, stackID, runID, note string, decision enums.RunReviewDecision) error {
	var runIDGQL *graphql.ID
	if runID != "" {
		runIDGQL = graphql.NewID(runID)
	}

	var mutation runReviewMutation
	variables := map[string]interface{}{
		"stack":    graphql.ID(stackID),
		"run":      runIDGQL,
		"decision": decision,
		"note":     note,
	}

	return authenticated.Client.Mutate(ctx, &mutation, variables)
}
