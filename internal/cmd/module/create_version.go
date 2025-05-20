package module

import (
	"context"
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func createVersionFunc() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		moduleID := cliCmd.String(flagModuleID.Name)
		forcedCommitSHA := cliCmd.String(flagCommitSHA.Name)
		forcedVersion := cliCmd.String(flagVersion.Name)

		var mutation struct {
			CreateModuleVersion struct {
				ID     string `graphql:"id"`
				Number string `graphql:"number"`
			} `graphql:"versionCreate(module: $module, commitSha: $commitSha, version: $version)"`
		}

		var version *graphql.String
		if forcedVersion != "" {
			version = graphql.NewString(graphql.String(forcedVersion))
		}
		var commitSha *graphql.String
		if forcedCommitSHA != "" {
			commitSha = graphql.NewString(graphql.String(forcedCommitSHA))
		}

		variables := map[string]interface{}{
			"module":    graphql.ID(moduleID),
			"commitSha": commitSha,
			"version":   version,
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return err
		}

		fmt.Printf("Module version %q has been successfully created\n", mutation.CreateModuleVersion.Number)

		return nil
	}
}
