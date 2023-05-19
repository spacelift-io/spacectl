package run_external_dependency

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type RunExternalDependencyStatus string

func markRunExternalDependencyAsFinished(cliCtx *cli.Context) error {
	externalDependencyID := cliCtx.String(flagRunExternalDependencyID.Name)
	status := cliCtx.String(flagStatus.Name)

	var mutation struct {
		RunExternalDependencyMarkAsFinished int `graphql:"runExternalDependencyMarkAsFinished(dependency: $dependency, status: $status)"`
	}

	if !slices.Contains([]string{"finished", "failed"}, status) {
		return fmt.Errorf("status must be one of: finished, failed")
	}

	variables := map[string]interface{}{
		"dependency": externalDependencyID,
		"status":     RunExternalDependencyStatus(strings.ToUpper(status)),
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Marked external dependencies for %d runs as %s\n", mutation.RunExternalDependencyMarkAsFinished, status)

	return nil
}
