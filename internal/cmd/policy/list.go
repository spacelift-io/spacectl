package policy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type listCommand struct{}

func (c *listCommand) list(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	var limit *uint
	if cliCtx.IsSet(cmd.FlagLimit.Name) {
		limit = internal.Ptr(cliCtx.Uint(cmd.FlagLimit.Name))
	}

	var search *string
	if cliCtx.IsSet(cmd.FlagSearch.Name) {
		search = internal.Ptr(cliCtx.String(cmd.FlagSearch.Name))
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.listTable(cliCtx, search, limit)
	case cmd.OutputFormatJSON:
		return c.listJSON(cliCtx, search, limit)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

func (c *listCommand) listTable(ctx *cli.Context, search *string, limit *uint) error {
	var first *int
	if limit != nil {
		first = internal.Ptr(int(*limit))
	}

	input := structs.SearchInput{
		First:          first,
		FullTextSearch: search,
		OrderBy: &structs.QueryOrder{
			Field:     "name",
			Direction: "DESC",
		},
	}

	policies, err := c.searchAllPolicies(ctx.Context, input)
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
			cmd.HumanizeUnixSeconds(b.UpdatedAt),
			strings.Join(b.Labels, ", "),
		}
		if ctx.Bool(cmd.FlagShowLabels.Name) {
			row = append(row, strings.Join(b.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

func (c *listCommand) listJSON(ctx *cli.Context, search *string, limit *uint) error {
	var first *int
	if limit != nil {
		first = internal.Ptr(int(*limit))
	}

	policies, err := c.searchAllPolicies(ctx.Context, structs.SearchInput{
		First:          first,
		FullTextSearch: search,
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(policies)
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
		First:          internal.Ptr(maxPageSize),
		FullTextSearch: input.FullTextSearch,
	}
	for {
		if !fetchAll {
			// Fetch exactly the number of items requested
			pageInput.First = internal.Ptr(slices.Min([]int{maxPageSize, limit - len(out)}))
		}

		result, err := searchPolicies(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Policies...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = internal.Ptr(result.PageInfo.EndCursor)
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
