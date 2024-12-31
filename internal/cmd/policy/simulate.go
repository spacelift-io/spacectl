package policy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type simulateCommand struct{}

func (c *simulateCommand) simulate(cliCtx *cli.Context) error {
	policyID := cliCtx.String(flagRequiredPolicyID.Name)
	input := cliCtx.String(flagSimulationInput.Name)

	parsedInput, err := parseInput(input)
	if err != nil {
		return err
	}

	b, found, err := getPolicyByID(cliCtx.Context, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to query for policy ID %q", policyID)
	}

	if !found {
		return fmt.Errorf("policy with ID %q not found", policyID)
	}

	var mutation struct {
		PolicySimulate string `graphql:"policySimulate(body: $body, input: $input, type: $type)"`
	}

	variables := map[string]interface{}{
		"body":  b.Body,
		"input": parsedInput,
		"type":  b.Type,
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
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

		var result map[string]interface{}
		if err := json.Unmarshal(fileContent, &result); err == nil {
			return string(fileContent), nil
		}
	} else {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(input), &result); err == nil {
			return input, nil
		}
	}

	return "", fmt.Errorf("input is neither a valid JSON nor a file path")
}
