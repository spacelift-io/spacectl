package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// PolicySimulateInput represents the input for simulating a policy.
type PolicySimulateInput struct {
	Body  string     `json:"body"`
	Input string     `json:"input"`
	Type  PolicyType `json:"type"`
}

type simulateCommand struct{}

func (c *simulateCommand) simulate(ctx context.Context, cliCmd *cli.Command) error {
	policyID := cliCmd.String(flagRequiredPolicyID.Name)
	input := cliCmd.String(flagSimulationInput.Name)

	parsedInput, err := parseInput(input)
	if err != nil {
		return err
	}

	b, found, err := getPolicyByID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to query for policy ID %q", policyID)
	}

	if !found {
		return fmt.Errorf("policy with ID %q not found", policyID)
	}

	var mutation struct {
		PolicySimulate string `graphql:"policySimulatev2(input: $input)"`
	}

	variables := map[string]any{
		"input": PolicySimulateInput{
			Body:  b.Body,
			Input: parsedInput,
			Type:  b.Type,
		},
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	return cmd.OutputJSON(mutation.PolicySimulate)
}

func parseInput(input string) (string, error) {
	if _, err := os.Stat(input); err == nil {
		fileContent, err := os.ReadFile(input)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(fileContent, &result); err == nil {
			return string(fileContent), nil
		}
	} else {
		var result map[string]any
		if err := json.Unmarshal([]byte(input), &result); err == nil {
			return input, nil
		}
	}

	return "", fmt.Errorf("input is neither a valid JSON nor a file path")
}
