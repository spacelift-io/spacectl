package stack

import (
	"fmt"
	"strconv"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type output struct {
	ID          string `graphql:"id" json:"id,omitempty"`
	Sensitive   bool   `graphql:"sensitive" json:"sensitive,omitempty"`
	Description string `graphql:"description" json:"description,omitempty"`
	Value       string `graphql:"value" json:"value,omitempty"`
}

type showOutputsQuery struct {
	Stack *struct {
		ID      string   `graphql:"id" json:"id,omitempty"`
		Name    string   `graphql:"name" json:"name,omitempty"`
		Outputs []output `graphql:"outputs" json:"outputs,omitempty"`
	} `graphql:"stack(id: $stackId)" json:"stacks,omitempty"`
}

type showOutputsStackCommand struct{}

func (c *showOutputsStackCommand) showOutputs(cliCtx *cli.Context) error {
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	outputID := cliCtx.String(flagOutputID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	var query showOutputsQuery
	variables := map[string]interface{}{
		"stackId": graphql.ID(stackID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return errors.Wrapf(err, "failed to query for stack ID %q", stackID)
	}

	if query.Stack == nil {
		return fmt.Errorf("stack ID %q not found", stackID)
	}

	var outputs []output
	if outputID != "" {
		for _, output := range query.Stack.Outputs {
			if output.ID == outputID {
				outputs = append(outputs, output)
			}
		}
	} else {
		outputs = query.Stack.Outputs
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showOutputsTable(outputs)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(outputs)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *showOutputsStackCommand) showOutputsTable(outputs []output) error {
	tableData := [][]string{{"Name", "Sensitive", "Value", "Description"}}
	for _, output := range outputs {
		tableData = append(tableData, []string{
			output.ID,
			strconv.FormatBool(output.Sensitive),
			output.Value,
			output.Description,
		})

	}
	return cmd.OutputTable(tableData, true)
}
