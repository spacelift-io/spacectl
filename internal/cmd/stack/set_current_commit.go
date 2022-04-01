package stack

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func setCurrentCommit(cliCtx *cli.Context) error {
	stackID := cliCtx.String(flagStackID.Name)

	var mutation struct {
		SetCurrentCommmit struct {
			TrackedCommit *struct {
				Hash    string `graphql:"hash"`
				Message string `graphql:"message"`
			} `graphql:"trackedCommit"`
		} `graphql:"stackSetCurrentCommit(id: $stack, sha: $sha)"`
	}

	variables := map[string]interface{}{
		"sha":   graphql.String(cliCtx.String(flagRequiredCommitSHA.Name)),
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

	_, err := fmt.Fprintf(os.Stdout, "Current commit set to %q: (SHA %s)", commit.Message, commit.Hash)

	return err
}
