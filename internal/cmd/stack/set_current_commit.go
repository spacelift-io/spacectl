package stack

import (
	"context"
	"errors"
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func setCurrentCommit(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	var mutation struct {
		SetCurrentCommmit struct {
			TrackedCommit *struct {
				Hash    string `graphql:"hash"`
				Message string `graphql:"message"`
			} `graphql:"trackedCommit"`
		} `graphql:"stackSetCurrentCommit(id: $stack, sha: $sha)"`
	}

	variables := map[string]interface{}{
		"sha":   cliCtx.String(flagRequiredCommitSHA.Name),
		"stack": graphql.ID(stackID),
	}

	ctx := context.Background()

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	commit := mutation.SetCurrentCommmit.TrackedCommit
	if commit == nil {
		return errors.New("no tracked commit set on the Stack")
	}

	_, err = fmt.Printf("Current commit set to %q: (SHA %s)\n", commit.Message, commit.Hash)

	return err
}
