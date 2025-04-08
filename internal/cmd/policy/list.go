package policy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type listCommand struct{}

func (c *listCommand) list(ctx context.Context, cmd *cli.Command) error {
	outputFormat, err := internalCmd.GetOutputFormat(cmd)
	if err != nil {
		return err
	}

	var limit *uint64
	if cmd.IsSet(internalCmd.FlagLimit.Name) {
		limit = internal.Ptr(cmd.Uint(internalCmd.FlagLimit.Name))
	}

	var search *string
	if cmd.IsSet(internalCmd.FlagSearch.Name) {
		search = internal.Ptr(cmd.String(internalCmd.FlagSearch.Name))
	}

	switch outputFormat {
	case internalCmd.OutputFormatTable:
		return c.listTable(ctx, cmd, search, limit)
	case internalCmd.OutputFormatJSON:
		return c.listJSON(ctx, search, limit)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *listCommand) listTable(ctx context.Context, cmd *cli.Command, search *string, limit *uint64) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = graphql.NewString(graphql.String(*search))
	}

	input := structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
		OrderBy: &structs.QueryOrder{
			Field:     "name",
			Direction: "DESC",
		},
	}

	policies, err := c.searchAllPolicies(ctx, input)
	if err != nil {
		return err
	}

	columns := []string{"Name", "ID", "Description", "Type", "Space", "Updated At", "Labels"}
	tableData := [][]string{columns}

	for _, b := range policies {
		row := []string{
			b.Name,
			b.ID,
			b.Description,
			b.Type,
			b.Space.ID,
			internalCmd.HumanizeUnixSeconds(b.UpdatedAt),
			strings.Join(b.Labels, ", "),
		}
		if cmd.Bool(internalCmd.FlagShowLabels.Name) {
			row = append(row, strings.Join(b.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return internalCmd.OutputTable(tableData, true)
}

func (c *listCommand) listJSON(ctx context.Context, search *string, limit *uint64) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = graphql.NewString(graphql.String(*search))
	}

	policies, err := c.searchAllPolicies(ctx, structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
	})
	if err != nil {
		return err
	}

	return internalCmd.OutputJSON(policies)
}

func (c *listCommand) searchAllPolicies(ctx context.Context, input structs.SearchInput) ([]policyNode, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []policyNode{}
	pageInput := structs.SearchInput{
		First:          graphql.NewInt(maxPageSize),
		FullTextSearch: input.FullTextSearch,
	}
	for {
		if !fetchAll {
			// Fetch exactly the number of items requested
			pageInput.First = graphql.NewInt(
				//nolint: gosec
				graphql.Int(
					slices.Min([]int{maxPageSize, limit - len(out)}),
				),
			)
		}

		result, err := searchPolicies(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Policies...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = graphql.NewString(graphql.String(result.PageInfo.EndCursor))
		} else {
			break
		}
	}

	return out, nil
}

type policyNode struct {
	ID          string `graphql:"id" json:"id"`
	Name        string `graphql:"name" json:"name"`
	Description string `graphql:"description" json:"description"`
	Body        string `graphql:"body" json:"body"`
	Space       struct {
		ID          string `graphql:"id" json:"id"`
		Name        string `graphql:"name" json:"name"`
		AccessLevel string `graphql:"accessLevel" json:"accessLevel"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	CreatedAt int      `graphql:"createdAt" json:"createdAt"`
	UpdatedAt int      `graphql:"updatedAt" json:"updatedAt"`
	Type      string   `graphql:"type" json:"type"`
	Labels    []string `graphql:"labels" json:"labels"`
}

type searchPoliciesResult struct {
	Policies []policyNode
	PageInfo structs.PageInfo
}

func searchPolicies(ctx context.Context, input structs.SearchInput) (searchPoliciesResult, error) {
	var query struct {
		SearchPoliciesOutput struct {
			Edges []struct {
				Node policyNode `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchPolicies(input: $input)"`
	}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		map[string]interface{}{"input": input},
	); err != nil {
		return searchPoliciesResult{}, errors.Wrap(err, "failed search for policies")
	}

	nodes := make([]policyNode, 0)
	for _, q := range query.SearchPoliciesOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchPoliciesResult{
		Policies: nodes,
		PageInfo: query.SearchPoliciesOutput.PageInfo,
	}, nil
}
