package policy

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type policyEvaluationSample struct {
	Input string `graphql:"input" json:"input"`
	Body  string `graphql:"body" json:"body"`
}

type sampleCommand struct{}

func (c *sampleCommand) show(ctx context.Context, cmd *cli.Command) error {
	policyID := cmd.String(flagRequiredPolicyID.Name)
	key := cmd.String(flagRequiredSampleKey.Name)

	b, err := c.getSamplesPolicyByID(ctx, policyID, key)
	if err != nil {
		return errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	return internalCmd.OutputJSON(b)
}

func (c *sampleCommand) getSamplesPolicyByID(ctx context.Context, policyID, key string) (policyEvaluationSample, error) {
	var query struct {
		Policy struct {
			Sample policyEvaluationSample `graphql:"evaluationSample(key: $key)"`
		} `graphql:"policy(id: $policyId)"`
	}

	variables := map[string]interface{}{
		"policyId": graphql.ID(policyID),
		"key":      graphql.String(key),
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return policyEvaluationSample{}, errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	return query.Policy.Sample, nil
}
