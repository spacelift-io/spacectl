package module

import (
	"fmt"
	"slices"

	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

const (
	maxSearchModulesPageSize = 50
)

func listModules() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			m, err := getModules(cliCtx, 20)
			if err != nil {
				return err
			}
			return listModulesTable(m)
		case cmd.OutputFormatJSON:
			m, err := getModules(cliCtx, 0)
			if err != nil {
				return err
			}
			return cmd.OutputJSON(m)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func getModules(cliCtx *cli.Context, limit int) ([]module, error) {
	if limit < 0 {
		return nil, errors.New("limit must be greater or equal to 0")
	}

	var cursor string
	var modules []module

	for {
		pageSize := 50
		if limit != 0 {
			pageSize = slices.Min([]int{maxSearchModulesPageSize, limit - len(modules)})
		}

		result, err := getSearchModules(cliCtx, cursor, pageSize)
		if err != nil {
			return nil, err
		}

		for _, edge := range result.Edges {
			modules = append(modules, edge.Node)
		}

		if result.PageInfo.HasNextPage && (limit == 0 || limit > len(modules)) {
			cursor = result.PageInfo.EndCursor
			continue
		}

		break
	}

	return modules, nil
}

func getSearchModules(cliCtx *cli.Context, cursor string, limit int) (searchModules, error) {
	if limit <= 0 || limit > maxSearchModulesPageSize {
		return searchModules{}, errors.New("limit must be between 1 and 50")
	}

	var query struct {
		SearchModules searchModules `graphql:"searchModules(input: $input)"`
	}

	var after *string
	if cursor != "" {
		after = internal.Ptr(cursor)
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{
		"input": structs.SearchInput{
			First: internal.Ptr(limit),
			After: after,
			OrderBy: &structs.QueryOrder{
				Field:     "starred",
				Direction: "DESC",
			},
		},
	}); err != nil {
		return searchModules{}, errors.Wrap(err, "failed to query list of modules")
	}

	return query.SearchModules, nil
}

func listModulesTable(modules []module) error {
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

	return cmd.OutputTable(tableData, true)
}

type searchModules struct {
	PageInfo structs.PageInfo    `graphql:"pageInfo"`
	Edges    []searchModulesEdge `graphql:"edges"`
}

type searchModulesEdge struct {
	Node module `graphql:"node"`
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
