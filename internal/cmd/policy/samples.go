package policy

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type policyEvaluation struct {
	ID                string `graphql:"id" json:"id"`
	EvaluationRecords []struct {
		Key       string `graphql:"key" json:"key"`
		Outcome   string `graphql:"outcome" json:"outcome"`
		Timestamp int    `graphql:"timestamp" json:"timestamp"`
	}
}

type samplesCommand struct{}

func (c *samplesCommand) list(ctx context.Context, cliCmd *cli.Command) error {
	policyID := cliCmd.String(flagRequiredPolicyID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	b, found, err := c.getSamplesPolicyByID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	if !found {
		return fmt.Errorf("policyEvaluation with ID %q not found", policyID)
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.samplesPolicyTable(b)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(b)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *samplesCommand) getSamplesPolicyByID(ctx context.Context, policyID string) (policyEvaluation, bool, error) {
	var query struct {
		Policy *policyEvaluation `graphql:"policy(id: $policyId)" json:"policy,omitempty"`
	}

	variables := map[string]interface{}{
		"policyId": graphql.ID(policyID),
	}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return policyEvaluation{}, false, errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	if query.Policy == nil {
		return policyEvaluation{}, false, nil
	}

	return *query.Policy, true, nil
}

func (c *samplesCommand) samplesPolicyTable(input policyEvaluation) error {
	tableData := [][]string{
		{"Key", "Outcome", "Timestamp"},
	}

	for _, record := range input.EvaluationRecords {
		tableData = append(tableData, []string{
			record.Key,
			record.Outcome,
			cmd.HumanizeUnixSeconds(record.Timestamp),
		})
	}

	if err := cmd.OutputTable(tableData, false); err != nil {
		return err
	}

	return nil
}
