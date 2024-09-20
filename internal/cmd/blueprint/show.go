package blueprint

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type showQuery struct {
	Blueprint *struct {
		ID          string `graphql:"id" json:"id,omitempty"`
		Deleted     bool   `graphql:"deleted" json:"deleted,omitempty"`
		Name        string `graphql:"name" json:"name,omitempty"`
		Description string `graphql:"description" json:"description,omitempty"`
		CreatedAt   int    `graphql:"createdAt" json:"createdAt,omitempty"`
		UpdatedAt   int    `graphql:"updatedAt" json:"updatedAt,omitempty"`
		State       string `graphql:"state" json:"state,omitempty"`
		Inputs      []struct {
			ID          string   `graphql:"id" json:"id,omitempty"`
			Name        string   `graphql:"name" json:"name,omitempty"`
			Default     string   `graphql:"default" json:"default,omitempty"`
			Description string   `graphql:"description" json:"description,omitempty"`
			Options     []string `graphql:"options" json:"options,omitempty"`
			Type        string   `graphql:"type" json:"type,omitempty"`
		}
		Space struct {
			ID          string `graphql:"id" json:"id,omitempty"`
			Name        string `graphql:"name" json:"name,omitempty"`
			AccessLevel string `graphql:"accessLevel" json:"accessLevel,omitempty"`
		}
		Labels      []string `graphql:"labels" json:"labels,omitempty"`
		RawTemplate string   `graphql:"rawTemplate" json:"rawTemplate,omitempty"`
	} `graphql:"blueprint(id: $blueprintId)" json:"blueprint,omitempty"`
}

type showCommand struct{}

func (c *showCommand) show(cliCtx *cli.Context) error {
	blueprintID := cliCtx.String(flagRequiredBlueprintID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	var query showQuery
	variables := map[string]interface{}{
		"blueprintId": graphql.ID(blueprintID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return errors.Wrapf(err, "failed to query for blueprint ID %q", blueprintID)
	}

	if query.Blueprint == nil {
		return fmt.Errorf("blueprint with ID %q not found", blueprintID)
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showBlueprintTable(query)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(query.Blueprint)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *showCommand) showBlueprintTable(query showQuery) error {
	c.outputBlueprintNameSection(query)

	if err := c.outputInputs(query); err != nil {
		return err
	}

	if err := c.outputSpace(query); err != nil {
		return err
	}

	c.outputRawTemplate(query)

	return nil
}

func (c *showCommand) outputBlueprintNameSection(query showQuery) {
	pterm.DefaultSection.WithLevel(1).Print(query.Blueprint.Name)

	if len(query.Blueprint.Labels) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Labels")
		pterm.DefaultParagraph.Println(fmt.Sprintf("[%s]", strings.Join(query.Blueprint.Labels, "], [")))
	}

	if query.Blueprint.Description != "" {
		pterm.DefaultSection.WithLevel(2).Println("Description")
		pterm.DefaultParagraph.Println(query.Blueprint.Description)
	}

	if query.Blueprint.CreatedAt != 0 {
		pterm.DefaultSection.WithLevel(2).Println("Created at")
		pterm.DefaultParagraph.Println(cmd.HumanizeUnixSeconds(query.Blueprint.CreatedAt))
	}

	if query.Blueprint.UpdatedAt != 0 {
		pterm.DefaultSection.WithLevel(2).Println("Updated at")
		pterm.DefaultParagraph.Println(cmd.HumanizeUnixSeconds(query.Blueprint.UpdatedAt))
	}

	if query.Blueprint.State != "" {
		pterm.DefaultSection.WithLevel(2).Println("State")
		pterm.DefaultParagraph.Println(cmd.HumanizeBlueprintState(query.Blueprint.State))
	}
}

func (c *showCommand) outputInputs(query showQuery) error {
	if len(query.Blueprint.Inputs) == 0 {
		return nil
	}

	pterm.DefaultSection.WithLevel(2).Println("Inputs")

	tableData := [][]string{{"Name", "ID", "Description", "Default", "Options", "Type"}}
	for _, input := range query.Blueprint.Inputs {

		tableData = append(tableData, []string{
			input.Name,
			input.ID,
			input.Description,
			input.Default,
			strings.Join(input.Options, ", "),
			input.Type,
		})
	}

	return cmd.OutputTable(tableData, true)
}

func (c *showCommand) outputSpace(query showQuery) error {
	pterm.DefaultSection.WithLevel(2).Println("Space")
	tableData := [][]string{
		{"Name", query.Blueprint.Space.Name},
		{"ID", query.Blueprint.Space.ID},
		{"Access Level", query.Blueprint.Space.AccessLevel},
	}

	return cmd.OutputTable(tableData, false)
}

func (c *showCommand) outputRawTemplate(query showQuery) {
	pterm.DefaultSection.WithLevel(2).Println("Template")

	pterm.Println(query.Blueprint.RawTemplate)
}
