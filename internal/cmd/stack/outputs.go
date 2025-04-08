package stack

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
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

func (c *showOutputsStackCommand) showOutputs(ctx context.Context, cmd *cli.Command) error {
	stackID, err := getStackID(ctx, cmd)
	if err != nil {
		return err
	}
	outputID := cmd.String(flagOutputID.Name)

	outputFormat, err := internalCmd.GetOutputFormat(cmd)
	if err != nil {
		return err
	}

	var query showOutputsQuery
	variables := map[string]interface{}{
		"stackId": graphql.ID(stackID),
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
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
	case internalCmd.OutputFormatTable:
		return c.showOutputsTable(outputs)
	case internalCmd.OutputFormatJSON:
		return internalCmd.OutputJSON(outputs)
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
	return internalCmd.OutputTable(tableData, true)
}
