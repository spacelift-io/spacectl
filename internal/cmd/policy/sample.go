package policy

import (
	"context"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type policyEvaluationSample struct {
	Input string `graphql:"input" json:"input"`
	Body  string `graphql:"body" json:"body"`
}

type sampleCommand struct{}

func (c *sampleCommand) show(cliCtx *cli.Context) error {
	policyID := cliCtx.String(flagRequiredPolicyID.Name)
	key := cliCtx.String(flagRequiredSampleKey.Name)

	b, err := c.getSamplesPolicyByID(cliCtx.Context, policyID, key)
	if err != nil {
		return errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	return cmd.OutputJSON(b)
}

func (c *sampleCommand) getSamplesPolicyByID(ctx context.Context, policyID, key string) (policyEvaluationSample, error) {
	var query struct {
		Policy struct {
			Sample policyEvaluationSample `graphql:"evaluationSample(key: $key)"`
		} `graphql:"policy(id: $policyId)"`
	}

	variables := map[string]interface{}{
		"policyId": graphql.ID(policyID),
		"key":      key,
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return policyEvaluationSample{}, errors.Wrapf(err, "failed to query for policyEvaluation ID %q", policyID)
	}

	return query.Policy.Sample, nil
}
