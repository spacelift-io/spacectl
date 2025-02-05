package blueprint

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

type blueprintInput struct {
	ID          string   `graphql:"id" json:"id,omitempty"`
	Name        string   `graphql:"name" json:"name,omitempty"`
	Default     string   `graphql:"default" json:"default,omitempty"`
	Description string   `graphql:"description" json:"description,omitempty"`
	Options     []string `graphql:"options" json:"options,omitempty"`
	Type        string   `graphql:"type" json:"type,omitempty"`
}

type blueprint struct {
	ID          string `graphql:"id" json:"id,omitempty"`
	Deleted     bool   `graphql:"deleted" json:"deleted,omitempty"`
	Name        string `graphql:"name" json:"name,omitempty"`
	Description string `graphql:"description" json:"description,omitempty"`
	CreatedAt   int    `graphql:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt   int    `graphql:"updatedAt" json:"updatedAt,omitempty"`
	State       string `graphql:"state" json:"state,omitempty"`
	Inputs      []blueprintInput
	Space       struct {
		ID          string `graphql:"id" json:"id,omitempty"`
		Name        string `graphql:"name" json:"name,omitempty"`
		AccessLevel string `graphql:"accessLevel" json:"accessLevel,omitempty"`
	}
	Labels      []string `graphql:"labels" json:"labels,omitempty"`
	RawTemplate string   `graphql:"rawTemplate" json:"rawTemplate,omitempty"`
}

type showCommand struct{}

func (c *showCommand) show(cliCtx *cli.Context) error {
	blueprintID := cliCtx.String(flagRequiredBlueprintID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	b, found, err := getBlueprintByID(cliCtx.Context, blueprintID)
	if err != nil {
		return errors.Wrapf(err, "failed to query for blueprint ID %q", blueprintID)
	}

	if !found {
		return fmt.Errorf("blueprint with ID %q not found", blueprintID)
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showBlueprintTable(b)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(b)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *showCommand) showBlueprintTable(b blueprint) error {
	c.outputBlueprintNameSection(b)

	if err := c.outputInputs(b); err != nil {
		return err
	}

	if err := c.outputSpace(b); err != nil {
		return err
	}

	c.outputRawTemplate(b)

	return nil
}

func (c *showCommand) outputBlueprintNameSection(b blueprint) {
	pterm.DefaultSection.WithLevel(1).Print(b.Name)

	if len(b.Labels) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Labels")
		pterm.DefaultParagraph.Println(fmt.Sprintf("[%s]", strings.Join(b.Labels, "], [")))
	}

	if b.Description != "" {
		pterm.DefaultSection.WithLevel(2).Println("Description")
		pterm.DefaultParagraph.Println(b.Description)
	}

	if b.CreatedAt != 0 {
		pterm.DefaultSection.WithLevel(2).Println("Created at")
		pterm.DefaultParagraph.Println(cmd.HumanizeUnixSeconds(b.CreatedAt))
	}

	if b.UpdatedAt != 0 {
		pterm.DefaultSection.WithLevel(2).Println("Updated at")
		pterm.DefaultParagraph.Println(cmd.HumanizeUnixSeconds(b.UpdatedAt))
	}

	if b.State != "" {
		pterm.DefaultSection.WithLevel(2).Println("State")
		pterm.DefaultParagraph.Println(cmd.HumanizeBlueprintState(b.State))
	}
}

func (c *showCommand) outputInputs(b blueprint) error {
	if len(b.Inputs) == 0 {
		return nil
	}

	pterm.DefaultSection.WithLevel(2).Println("Inputs")

	tableData := [][]string{{"Name", "ID", "Description", "Default", "Options", "Type"}}
	for _, input := range b.Inputs {

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

func (c *showCommand) outputSpace(b blueprint) error {
	pterm.DefaultSection.WithLevel(2).Println("Space")
	tableData := [][]string{
		{"Name", b.Space.Name},
		{"ID", b.Space.ID},
		{"Access Level", b.Space.AccessLevel},
	}

	return cmd.OutputTable(tableData, false)
}

func (c *showCommand) outputRawTemplate(b blueprint) {
	pterm.DefaultSection.WithLevel(2).Println("Template")

	pterm.Println(b.RawTemplate)
}

func getBlueprintByID(ctx context.Context, blueprintID string) (blueprint, bool, error) {
	var query struct {
		Blueprint *blueprint `graphql:"blueprint(id: $blueprintId)" json:"blueprint,omitempty"`
	}

	variables := map[string]interface{}{
		"blueprintId": graphql.ID(blueprintID),
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return blueprint{}, false, errors.Wrapf(err, "failed to query for blueprint ID %q", blueprintID)
	}

	if query.Blueprint == nil {
		return blueprint{}, false, nil
	}

	return *query.Blueprint, true, nil
}
