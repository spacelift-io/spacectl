package runexternaldependency

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func markRunExternalDependencyAsCompleted(ctx context.Context, cliCmd *cli.Command) error {
	externalDependencyID := cliCmd.String(flagRunExternalDependencyID.Name)
	status := cliCmd.String(flagStatus.Name)

	var mutation struct {
		RunExternalDependencyMarkAsFinished struct {
			Phantom bool `graphql:"phantom"`
		} `graphql:"runExternalDependencyMarkAsCompleted(dependency: $dependency, status: $status)"`
	}

	if !slices.Contains([]string{"finished", "failed", "skipped"}, status) {
		return fmt.Errorf("status must be one of: finished, failed, skipped")
	}

	type RunExternalDependencyStatus string
	variables := map[string]interface{}{
		"dependency": externalDependencyID,
		"status":     RunExternalDependencyStatus(strings.ToUpper(status)),
	}

	if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	fmt.Printf("Marked external dependency as %s\n", status)

	return nil
}
