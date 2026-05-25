package template

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const maxPageSize = 50

var _ cli.ActionFunc = listTemplates

type templateNode struct {
	ID               string   `graphql:"id" json:"id,omitempty"`
	Name             string   `graphql:"name" json:"name,omitempty"`
	Description      string   `graphql:"description" json:"description,omitempty"`
	Labels           []string `graphql:"labels" json:"labels,omitempty"`
	CreatedAt        int      `graphql:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt        int      `graphql:"updatedAt" json:"updatedAt,omitempty"`
	DeploymentsCount int      `graphql:"deploymentsCount" json:"deploymentsCount,omitempty"`
	Space            struct {
		ID          string `graphql:"id" json:"id,omitempty"`
		Name        string `graphql:"name" json:"name,omitempty"`
		AccessLevel string `graphql:"accessLevel" json:"accessLevel,omitempty"`
	} `graphql:"space" json:"space"`
}

type searchTemplatesResult struct {
	Templates []templateNode
	PageInfo  structs.PageInfo
}

type searchTemplatesOutput struct {
	Edges []struct {
		Node templateNode `graphql:"node"`
	} `graphql:"edges"`
	PageInfo structs.PageInfo `graphql:"pageInfo"`
}

func listTemplates(ctx context.Context, cliCmd *cli.Command) error {
	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	input := structs.SearchInput{
		OrderBy: &structs.QueryOrder{
			Field:     "name",
			Direction: "DESC",
		},
	}

	if cliCmd.IsSet(cmd.FlagLimit.Name) {
		//nolint:gosec // Flag value has been pre-validated.
		input.First = new(graphql.Int(cliCmd.Uint(cmd.FlagLimit.Name)))
	}

	if cliCmd.IsSet(cmd.FlagSearch.Name) {
		input.FullTextSearch = new(graphql.String(cliCmd.String(cmd.FlagSearch.Name)))
	}

	templates, err := searchAllTemplates(ctx, input)
	if err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return printTemplateTable(templates, cliCmd.Bool(cmd.FlagShowLabels.Name))
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(templates)
	default: // Shouldn't happen as outputFormat is pre-validated.
		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

// searchAllTemplates returns a list of templates based on the provided search input.
// input.First limits the total number of returned templates, if not provided all templates are returned.
func searchAllTemplates(ctx context.Context, input structs.SearchInput) ([]templateNode, error) {
	limit := 0
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := make([]templateNode, 0, maxPageSize)
	pageInput := structs.SearchInput{
		First:          new(graphql.Int(maxPageSize)),
		FullTextSearch: input.FullTextSearch,
	}

	for {
		if !fetchAll {
			// Fetch exactly the number of items requested
			//nolint:gosec // Flag value has been pre-validated.
			pageInput.First = new(graphql.Int(min(maxPageSize, limit-len(out))))
		}

		result, err := searchTemplates(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Templates...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = new(graphql.String(result.PageInfo.EndCursor))
		} else {
			break
		}
	}

	return out, nil
}

func searchTemplates(
	ctx context.Context, input structs.SearchInput,
) (searchTemplatesResult, error) {
	var query struct {
		SearchTemplatesOutput searchTemplatesOutput `graphql:"searchTemplates(input: $input)"`
	}

	if err := authenticated.Client().Query(
		ctx, &query, map[string]any{"input": input},
	); err != nil {
		return searchTemplatesResult{}, errors.Wrap(err, "failed search for templates")
	}

	nodes := make([]templateNode, 0, maxPageSize)
	for _, q := range query.SearchTemplatesOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchTemplatesResult{
		Templates: nodes,
		PageInfo:  query.SearchTemplatesOutput.PageInfo,
	}, nil
}

func printTemplateTable(templates []templateNode, showLabels bool) error {
	columns := []string{"Name", "ID", "Description", "Space", "Deployments", "Updated At"}
	if showLabels {
		columns = append(columns, "Labels")
	}

	tableData := [][]string{columns}
	for _, t := range templates {
		row := []string{
			t.Name,
			t.ID,
			t.Description,
			t.Space.Name,
			strconv.Itoa(t.DeploymentsCount),
			cmd.HumanizeUnixSeconds(t.UpdatedAt),
		}

		if showLabels {
			row = append(row, strings.Join(t.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}
