package stack

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
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

func lock(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	note := cliCtx.String(flagStackLockNote.Name)

	if nArgs := cliCtx.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation stackLockMutation
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
		"note":  note,
	}

	return authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)
}

func unlock(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	if nArgs := cliCtx.NArg(); nArgs != 0 {
		return fmt.Errorf("expected zero arguments but got %d", nArgs)
	}

	var mutation stackUnlockMutation
	variables := map[string]interface{}{
		"stack": graphql.ID(stackID),
	}

	return authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)
}
