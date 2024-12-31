package module

import (
	"fmt"

	"github.com/hasura/go-graphql-client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func createVersion(cliCtx *cli.Context) error {
	moduleID := cliCtx.String(flagModuleID.Name)
	forcedCommitSHA := cliCtx.String(flagCommitSHA.Name)
	forcedVersion := cliCtx.String(flagVersion.Name)

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

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Module version %q has been successfully created\n", mutation.CreateModuleVersion.Number)

	return nil
}
