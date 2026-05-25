package template

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type templateDetail struct {
	templateNode
	CanDelete     bool `graphql:"canDelete" json:"canDelete"`
	LatestVersion *struct {
		ID      string `graphql:"id" json:"id,omitempty"`
		Version string `graphql:"version" json:"version,omitempty"`
	} `graphql:"latestVersion" json:"latestVersion,omitempty"`
}

func showTemplate(ctx context.Context, cliCmd *cli.Command) error {
	templateID := cliCmd.String(flagRequiredTemplateID.Name)

	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	t, found, err := getTemplateByID(ctx, templateID)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("template with ID %q not found", templateID)
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return showTemplateTable(t)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(t)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func showTemplateTable(t templateDetail) error {
	outputTemplateNameSection(t)

	if err := outputTemplateDetails(t); err != nil {
		return err
	}

	return outputTemplateSpace(t)
}

func outputTemplateNameSection(t templateDetail) {
	pterm.DefaultSection.WithLevel(1).Print(t.Name)

	if len(t.Labels) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Labels")
		pterm.DefaultParagraph.Println(fmt.Sprintf("[%s]", strings.Join(t.Labels, "], [")))
	}

	if t.Description != "" {
		pterm.DefaultSection.WithLevel(2).Println("Description")
		pterm.DefaultParagraph.Println(t.Description)
	}
}

func outputTemplateDetails(t templateDetail) error {
	pterm.DefaultSection.WithLevel(2).Println("Details")

	latestVersion := "none"
	if t.LatestVersion != nil && t.LatestVersion.Version != "" {
		latestVersion = t.LatestVersion.Version
	}

	tableData := [][]string{
		{"ID", t.ID},
		{"Created At", cmd.HumanizeUnixSeconds(t.CreatedAt)},
		{"Updated At", cmd.HumanizeUnixSeconds(t.UpdatedAt)},
		{"Deployments", strconv.Itoa(t.DeploymentsCount)},
		{"Latest Version", latestVersion},
		{"Can Delete", fmt.Sprint(t.CanDelete)},
	}

	return cmd.OutputTable(tableData, false)
}

func outputTemplateSpace(t templateDetail) error {
	pterm.DefaultSection.WithLevel(2).Println("Space")
	tableData := [][]string{
		{"Name", t.Space.Name},
		{"ID", t.Space.ID},
		{"Access Level", t.Space.AccessLevel},
	}

	return cmd.OutputTable(tableData, false)
}

func getTemplateByID(ctx context.Context, templateID string) (templateDetail, bool, error) {
	var query struct {
		Template *templateDetail `graphql:"template(id: $templateId)" json:"template,omitempty"`
	}

	variables := map[string]any{
		"templateId": graphql.ID(templateID),
	}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return templateDetail{}, false, errors.Wrapf(err, "failed to query for template ID %q", templateID)
	}

	if query.Template == nil {
		return templateDetail{}, false, nil
	}

	return *query.Template, true, nil
}
