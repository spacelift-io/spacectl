package blueprint

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

func listBlueprints() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		outputFormat, err := internalCmd.GetOutputFormat(cmd)
		if err != nil {
			return err
		}

		var limit *uint
		if cmd.IsSet(internalCmd.FlagLimit.Name) {
			limit = internal.Ptr(uint(cmd.Uint(internalCmd.FlagLimit.Name)))
		}

		var search *string
		if cmd.IsSet(internalCmd.FlagSearch.Name) {
			search = internal.Ptr(cmd.String(internalCmd.FlagSearch.Name))
		}

		switch outputFormat {
		case internalCmd.OutputFormatTable:
			err := listBlueprintsTable(ctx, cmd, search, limit)
			return err
		case internalCmd.OutputFormatJSON:
			err := listBlueprintsJSON(ctx, search, limit)
			return err
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listBlueprintsJSON(
	ctx context.Context,
	search *string,
	limit *uint,
) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = graphql.NewString(graphql.String(*search))
	}

	blueprints, err := searchAllBlueprints(ctx, structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
	})
	if err != nil {
		return err
	}

	return internalCmd.OutputJSON(blueprints)
}

func listBlueprintsTable(
	ctx context.Context,
	cmd *cli.Command,
	search *string,
	limit *uint,
) error {
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

	blueprints, err := searchAllBlueprints(ctx, input)
	if err != nil {
		return err
	}

	columns := []string{"Name", "ID", "Description", "State", "Space", "Updated At"}
	if cmd.Bool(internalCmd.FlagShowLabels.Name) {
		columns = append(columns, "Labels")
	}

	tableData := [][]string{columns}
	for _, b := range blueprints {
		row := []string{
			b.Name,
			b.ID,
			b.Description,
			internalCmd.HumanizeBlueprintState(b.State),
			b.Space.Name,
			internalCmd.HumanizeUnixSeconds(b.UpdatedAt),
		}
		if cmd.Bool(internalCmd.FlagShowLabels.Name) {
			row = append(row, strings.Join(b.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return internalCmd.OutputTable(tableData, true)
}

// searchAllBlueprints returns a list of stacks based on the provided search input.
// input.First limits the total number of returned stacks, if not provided all stacks are returned.
func searchAllBlueprints(ctx context.Context, input structs.SearchInput) ([]blueprintNode, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []blueprintNode{}
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

		result, err := searchBlueprints(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Blueprints...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = graphql.NewString(graphql.String(result.PageInfo.EndCursor))
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
