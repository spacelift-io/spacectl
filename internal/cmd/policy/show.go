package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type PolicyType string

type policy struct {
	ID          string `graphql:"id" json:"id"`
	Name        string `graphql:"name" json:"name"`
	Description string `graphql:"description" json:"description"`
	Body        string `graphql:"body" json:"body"`
	Space       struct {
		ID          string `graphql:"id" json:"id"`
		Name        string `graphql:"name" json:"name"`
		AccessLevel string `graphql:"accessLevel" json:"accessLevel"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	CreatedAt int        `graphql:"createdAt" json:"createdAt"`
	UpdatedAt int        `graphql:"updatedAt" json:"updatedAt"`
	Type      PolicyType `graphql:"type" json:"type"`
	Labels    []string   `graphql:"labels" json:"labels"`
}

type showCommand struct{}

func (c *showCommand) show(cliCtx *cli.Context) error {
	policyID := cliCtx.String(flagRequiredPolicyID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCtx)
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

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showPolicyTable(b)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(b)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func getPolicyByID(ctx context.Context, policyID string) (policy, bool, error) {
	var query struct {
		Policy *policy `graphql:"policy(id: $policyId)" json:"policy,omitempty"`
	}

	variables := map[string]interface{}{
		"policyId": graphql.ID(policyID),
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return policy{}, false, errors.Wrapf(err, "failed to query for policy ID %q", policyID)
	}

	if query.Policy == nil {
		return policy{}, false, nil
	}

	return *query.Policy, true, nil
}

func (c *showCommand) showPolicyTable(input policy) error {
	pterm.DefaultSection.WithLevel(2).Println("Policy")

	tableData := [][]string{
		{"Name", input.Name},
		{"ID", input.ID},
		{"Description", input.Description},
		{"Type", string(input.Type)},
		{"Space", input.Space.ID},
		{"Labels", strings.Join(input.Labels, ", ")},
	}

	if err := cmd.OutputTable(tableData, false); err != nil {
		return err
	}

	pterm.DefaultSection.WithLevel(2).Println("Body")

	pterm.Println(input.Body)

	return nil
}
