package blueprint

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

func listBlueprints() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
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
			return listBlueprintsTable(cliCtx, search, limit)
		case cmd.OutputFormatJSON:
			return listBlueprintsJSON(cliCtx, search, limit)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listBlueprintsJSON(
	ctx *cli.Context,
	search *string,
	limit *uint,
) error {
	var first *int
	if limit != nil {
		first = internal.Ptr(int(*limit))
	}

	blueprints, err := searchAllBlueprints(ctx.Context, structs.SearchInput{
		First:          first,
		FullTextSearch: search,
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(blueprints)
}

func listBlueprintsTable(
	ctx *cli.Context,
	search *string,
	limit *uint,
) error {
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

	blueprints, err := searchAllBlueprints(ctx.Context, input)
	if err != nil {
		return err
	}

	columns := []string{"Name", "ID", "Description", "State", "Space", "Updated At"}
	if ctx.Bool(cmd.FlagShowLabels.Name) {
		columns = append(columns, "Labels")
	}

	tableData := [][]string{columns}
	for _, b := range blueprints {
		row := []string{
			b.Name,
			b.ID,
			b.Description,
			cmd.HumanizeBlueprintState(b.State),
			b.Space.Name,
			cmd.HumanizeUnixSeconds(b.UpdatedAt),
		}
		if ctx.Bool(cmd.FlagShowLabels.Name) {
			row = append(row, strings.Join(b.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

// searchAllBlueprints returns a list of stacks based on the provided search input.
// input.First limits the total number of returned stacks, if not provided all stacks are returned.
func searchAllBlueprints(ctx context.Context, input structs.SearchInput) ([]blueprintNode, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = *input.First
	}
	fetchAll := limit == 0

	out := []blueprintNode{}
	pageInput := structs.SearchInput{
		First:          internal.Ptr(maxPageSize),
		FullTextSearch: input.FullTextSearch,
	}
	for {
		if !fetchAll {
			// Fetch exactly the number of items requested
			pageInput.First = internal.Ptr(slices.Min([]int{maxPageSize, limit - len(out)}))
		}

		result, err := searchBlueprints(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Blueprints...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = internal.Ptr(result.PageInfo.EndCursor)
		} else {
			break
		}
	}

	return out, nil
}

type blueprintNode struct {
	ID          string   `graphql:"id" json:"id,omitempty"`
	Name        string   `graphql:"name" json:"name,omitempty"`
	State       string   `graphql:"state" json:"state,omitempty"`
	Description string   `graphql:"description" json:"description,omitempty"`
	Labels      []string `graphql:"labels" json:"labels,omitempty"`
	CreatedAt   int      `graphql:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt   int      `graphql:"updatedAt" json:"updatedAt,omitempty"`
	RawTemplate string   `graphql:"rawTemplate" json:"rawTemplate,omitempty"`
	Inputs      []struct {
		ID      string   `graphql:"id" json:"id,omitempty"`
		Name    string   `graphql:"name" json:"name,omitempty"`
		Default string   `graphql:"default" json:"default,omitempty"`
		Options []string `graphql:"options" json:"options,omitempty"`
		Type    string   `graphql:"type" json:"type,omitempty"`
	} `graphql:"inputs" json:"inputs,omitempty"`
	Space struct {
		ID          string `graphql:"id" json:"id,omitempty"`
		Name        string `graphql:"name" json:"name,omitempty"`
		AccessLevel string `graphql:"accessLevel" json:"accessLevel,omitempty"`
	} `graphql:"space" json:"space,omitempty"`
}

type searchBlueprintsResult struct {
	Blueprints []blueprintNode
	PageInfo   structs.PageInfo
}

func searchBlueprints(ctx context.Context, input structs.SearchInput) (searchBlueprintsResult, error) {
	var query struct {
		SearchBlueprintsOutput struct {
			Edges []struct {
				Node blueprintNode `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchBlueprints(input: $input)"`
	}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		map[string]interface{}{"input": input},
	); err != nil {
		return searchBlueprintsResult{}, errors.Wrap(err, "failed search for blueprints")
	}

	nodes := make([]blueprintNode, 0)
	for _, q := range query.SearchBlueprintsOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchBlueprintsResult{
		Blueprints: nodes,
		PageInfo:   query.SearchBlueprintsOutput.PageInfo,
	}, nil
}
