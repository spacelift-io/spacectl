package module

import (
	"fmt"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func createVersion(cliCtx *cli.Context) error {
	moduleID := cliCtx.String(flagModuleID.Name)
	commitSHA := cliCtx.String(flagCommitSHA.Name)
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

	variables := map[string]interface{}{
		"module":    graphql.ID(moduleID),
		"commitSha": graphql.String(commitSHA),
		"version":   version,
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Module version %q has been successfully created\n", mutation.CreateModuleVersion.Number)

	return nil
}
