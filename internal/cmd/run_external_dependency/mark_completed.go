package runexternaldependency

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func markRunExternalDependencyAsCompleted(cliCtx *cli.Context) error {
	externalDependencyID := cliCtx.String(flagRunExternalDependencyID.Name)
	status := cliCtx.String(flagStatus.Name)

	var mutation struct {
		RunExternalDependencyMarkAsFinished struct {
			Phantom bool `graphql:"phantom"`
		} `graphql:"runExternalDependencyMarkAsCompleted(dependency: $dependency, status: $status)"`
	}

	if !slices.Contains([]string{"finished", "failed"}, status) {
		return fmt.Errorf("status must be one of: finished, failed")
	}

	type RunExternalDependencyStatus string
	variables := map[string]interface{}{
		"dependency": externalDependencyID,
		"status":     RunExternalDependencyStatus(strings.ToUpper(status)),
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Marked external dependency as %s\n", status)

	return nil
}
