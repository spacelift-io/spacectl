package stack

import (
	"context"
	"errors"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func setCurrentCommit(ctx context.Context, cliCmd *cli.Command) error {
	stackID, err := getStackID(ctx, cliCmd)
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
		"sha":   graphql.String(cliCmd.String(flagRequiredCommitSHA.Name)),
		"stack": graphql.ID(stackID),
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	commit := mutation.SetCurrentCommmit.TrackedCommit
	if commit == nil {
		return errors.New("no tracked commit set on the Stack")
	}

	_, err = fmt.Printf("Current commit set to %q: (SHA %s)\n", commit.Message, commit.Hash)

	return err
}
