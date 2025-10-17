package stack

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type stackLockMutation struct {
	Stack struct {
		ID string `graphql:"id"`
	} `graphql:"stackLock(id: $stack, note: $note)"`
}
type stackUnlockMutation struct {
	Stack struct {
		ID string `graphql:"id"`
	} `graphql:"stackUnlock(id: $stack)"`
}

var flagStackLockNote = &cli.StringFlag{
	Name:     "note",
	Usage:    "Description of why the lock was acquired.",
	Required: false,
}

func lock(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	note := cliCmd.String(flagStackLockNote.Name)

	if nArgs := cliCmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation stackLockMutation
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"note":  graphql.String(note),
	}

	return authenticated.Client().Mutate(ctx, &mutation, variables)
}

func unlock(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}

	if nArgs := cliCmd.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation stackUnlockMutation
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	return authenticated.Client().Mutate(ctx, &mutation, variables)
}
