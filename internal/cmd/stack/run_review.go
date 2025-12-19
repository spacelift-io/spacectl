package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/enums"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
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

func runApprove(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd, nil)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)
	note := cliCmd.String(flagRunReviewNote.Name)

	if nArgs := cliCmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	return addRunReview(ctx, stackID, runID, note, enums.RunReviewDecisionApprove)
}

func runReject(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd, nil)
	if err != nil {
		return err
	}
	runID := cliCmd.String(flagRequiredRun.Name)
	note := cliCmd.String(flagRunReviewNote.Name)

	if nArgs := cliCmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	return addRunReview(ctx, stackID, runID, note, enums.RunReviewDecisionReject)
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
		"note":     graphql.String(note),
	}

	return authenticated.Client().Mutate(ctx, &mutation, variables)
}
