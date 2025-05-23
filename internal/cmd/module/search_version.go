package module

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const (
	maxSearchModuleVersionsPageSize = 50
	moduleVersionsTableLimit        = 20
	moduleVersionsJSONLimit         = 0 // no limit
)

func listVersions() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		outputFormat, err := cmd.GetOutputFormat(cliCmd)
		if err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			versions, err := getModuleVersions(ctx, cliCmd, moduleVersionsTableLimit)
			if err != nil {
				return err
			}

			return formatModuleVersionsTable(versions)
		case cmd.OutputFormatJSON:
			versions, err := getModuleVersions(ctx, cliCmd, moduleVersionsJSONLimit)
			if err != nil {
				return err
			}

			return formatModuleVersionsJSON(versions)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func getModuleVersions(ctx context.Context, cliCmd *cli.Command, limit int) ([]version, error) {
	if limit < 0 {
		return nil, errors.New("limit must be greater or equal to 0")
	}

	var cursor string
	var versions []version

	fetchAll := limit == 0

	var pageSize int

	for {
		if fetchAll {
			pageSize = maxSearchModuleVersionsPageSize
		} else {
			pageSize = slices.Min([]int{maxSearchModuleVersionsPageSize, limit - len(versions)})
		}

		result, err := getSearchModuleVersions(ctx, cliCmd, cursor, pageSize)
		if err != nil {
			return nil, err
		}

		for _, edge := range result.Edges {
			versions = append(versions, edge.Node)
		}

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(versions)) {
			cursor = result.PageInfo.EndCursor
		} else {
			break
		}
	}

	return versions, nil
}

func getSearchModuleVersions(ctx context.Context, cliCmd *cli.Command, cursor string, limit int) (searchModuleVersions, error) {
	if limit <= 0 || limit > maxSearchModuleVersionsPageSize {
		return searchModuleVersions{}, errors.New("limit must be between 1 and 50")
	}

	var query struct {
		Module struct {
			SearchModuleVersions searchModuleVersions `graphql:"searchModuleVersions(input: $input)"`
		} `graphql:"module(id: $id)"`
	}

	var after *graphql.String
	if cursor != "" {
		after = graphql.NewString(graphql.String(cursor))
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{
		"id": cliCmd.String(flagModuleID.Name),
		"input": structs.SearchInput{
			First: graphql.NewInt(graphql.Int(int32(limit))), //nolint: gosec
			After: after,
			OrderBy: &structs.QueryOrder{
				Field:     "createdAt",
				Direction: "DESC",
			},
			Predicates: &[]structs.QueryPredicate{
				{
					Field:   "state",
					Exclude: true,
					Constraint: structs.QueryFieldConstraint{
						EnumEquals: &[]graphql.String{
							"FAILED",
						},
					},
				},
			},
		},
	}); err != nil {
		return searchModuleVersions{}, errors.Wrap(err, "failed to query list of modules")
	}

	return query.Module.SearchModuleVersions, nil
}

func formatModuleVersionsJSON(versions []version) error {
	return cmd.OutputJSON(versions)
}

func formatModuleVersionsTable(versions []version) error {
	columns := []string{"ID", "Author", "Message", "Number", "State", "Tests", "Timestamp"}
	tableData := [][]string{columns}

	for _, v := range versions {
		tableData = append(tableData, []string{
			v.ID,
			v.Commit.AuthorName,
			v.Commit.Message,
			v.Number,
			v.State,
			fmt.Sprintf("%d", v.VersionCount),
			fmt.Sprintf("%d", v.Commit.Timestamp),
		})
	}

	return cmd.OutputTable(tableData, true)
}

type searchModuleVersions struct {
	PageInfo structs.PageInfo          `graphql:"pageInfo"`
	Edges    []searchModuleVersionEdge `graphql:"edges"`
}

type searchModuleVersionEdge struct {
	Node version `graphql:"node"`
}

type version struct {
	ID     string `json:"id" graphql:"id"`
	Commit struct {
		AuthorLogin string `json:"authorLogin" graphql:"authorLogin"`
		AuthorName  string `json:"authorName" graphql:"authorName"`
		Hash        string `json:"hash" graphql:"hash"`
		Message     string `json:"message" graphql:"message"`
		Timestamp   int    `json:"timestamp" graphql:"timestamp"`
		URL         string `json:"url" graphql:"url"`
	} `json:"commit" graphql:"commit"`
	DownloadLink interface{} `json:"downloadLink" graphql:"downloadLink"`
	Number       string      `json:"number" graphql:"number"`
	SourceURL    string      `json:"sourceURL" graphql:"sourceURL"`
	State        string      `json:"state" graphql:"state"`
	VersionCount int         `json:"versionCount" graphql:"versionCount"`
	Yanked       bool        `json:"yanked" graphql:"yanked"`
}
