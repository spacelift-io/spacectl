package module

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func listModules() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		outputFormat, err := cmd.GetOutputFormat(cliCmd)
		if err != nil {
			return err
		}

		var limit *uint
		if cliCmd.IsSet(cmd.FlagLimit.Name) {
			limit = internal.Ptr(cliCmd.Uint(cmd.FlagLimit.Name))
		}

		var search *string
		if cliCmd.IsSet(cmd.FlagSearch.Name) {
			search = internal.Ptr(cliCmd.String(cmd.FlagSearch.Name))
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return listModulesTable(ctx, search, limit)
		case cmd.OutputFormatJSON:
			return listModulesJSON(ctx, search, limit)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listModulesJSON(ctx context.Context, search *string, limit *uint) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = graphql.NewString(graphql.String(*search))
	}

	modules, err := searchAllModules(ctx, structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(modules)
}

func listModulesTable(ctx context.Context, search *string, limit *uint) error {
	const defaultLimit = 20

	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	} else {
		first = graphql.NewInt(graphql.Int(defaultLimit))
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

	modules, err := searchAllModules(ctx, input)
	if err != nil {
		return err
	}
	columns := []string{"Name", "ID", "Current Version", "Number", "State", "Yanked"}

	tableData := [][]string{columns}
	for _, module := range modules {
		row := []string{
			module.Name,
			module.ID,
			module.Current.ID,
			module.Current.Number,
			module.Current.State,
			fmt.Sprintf("%t", module.Current.Yanked),
		}

		tableData = append(tableData, row)
	}

	if err := cmd.OutputTable(tableData, true); err != nil {
		return err
	}

	if limit == nil {
		fmt.Printf("Showing first %d modules. Use --limit to show more or less.\n", defaultLimit)
	}

	return nil
}

func searchAllModules(ctx context.Context, input structs.SearchInput) ([]module, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []module{}
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

		result, err := searchModules(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Modules...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = graphql.NewString(graphql.String(result.PageInfo.EndCursor))
		} else {
			break
		}
	}

	return out, nil
}

type searchModulesResult struct {
	Modules  []module
	PageInfo structs.PageInfo
}

type module struct {
	ID                  string   `json:"id" graphql:"id"`
	Administrative      bool     `json:"administrative" graphql:"administrative"`
	APIHost             string   `json:"apiHost" graphql:"apiHost"`
	Branch              string   `json:"branch" graphql:"branch"`
	Description         string   `json:"description" graphql:"description"`
	CanWrite            bool     `json:"canWrite" graphql:"canWrite"`
	IsDisabled          bool     `json:"isDisabled" graphql:"isDisabled"`
	CreatedAt           int      `json:"createdAt" graphql:"createdAt"`
	Labels              []string `json:"labels" graphql:"labels"`
	Name                string   `json:"name" graphql:"name"`
	Namespace           string   `json:"namespace" graphql:"namespace"`
	ProjectRoot         string   `json:"projectRoot" graphql:"projectRoot"`
	Provider            string   `json:"provider" graphql:"provider"`
	Repository          string   `json:"repository" graphql:"repository"`
	TerraformProvider   string   `json:"terraformProvider" graphql:"terraformProvider"`
	LocalPreviewEnabled bool     `json:"localPreviewEnabled,omitempty" graphql:"localPreviewEnabled"`
	Current             struct {
		ID     string `json:"id" graphql:"id"`
		Number string `json:"number" graphql:"number"`
		State  string `json:"state" graphql:"state"`
		Yanked bool   `json:"yanked" graphql:"yanked"`
	} `json:"current" graphql:"current"`
	SpaceDetails struct {
		ID          string `json:"id" graphql:"id"`
		Name        string `json:"name" graphql:"name"`
		AccessLevel string `json:"accessLevel" graphql:"accessLevel"`
	} `json:"spaceDetails" graphql:"spaceDetails"`
	WorkerPool struct {
		ID   string `graphql:"id" json:"id,omitempty"`
		Name string `graphql:"name" json:"name,omitempty"`
	} `graphql:"workerPool" json:"workerPool,omitempty"`
	Starred bool `json:"starred" graphql:"starred"`
}

func searchModules(ctx context.Context, input structs.SearchInput) (searchModulesResult, error) {
	var query struct {
		SearchModulesOutput struct {
			Edges []struct {
				Node module `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchModules(input: $input)"`
	}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		map[string]interface{}{"input": input},
	); err != nil {
		return searchModulesResult{}, errors.Wrap(err, "failed search for modules")
	}

	nodes := make([]module, 0)
	for _, q := range query.SearchModulesOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchModulesResult{
		Modules:  nodes,
		PageInfo: query.SearchModulesOutput.PageInfo,
	}, nil
}
