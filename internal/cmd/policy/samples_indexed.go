package policy

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

func samplesIndexed() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		policyID := cliCmd.String(flagRequiredPolicyID.Name)

		outputFormat, err := cmd.GetOutputFormat(cliCmd)
		if err != nil {
			return err
		}

		var limit *uint
		if cliCmd.IsSet(cmd.FlagLimit.Name) {
			limit = internal.Ptr(cliCmd.Uint(cmd.FlagLimit.Name))
		}

		var outcome *string
		if cliCmd.IsSet(flagOutcomeFilter.Name) {
			outcome = internal.Ptr(cliCmd.String(flagOutcomeFilter.Name))
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return samplesIndexedTable(ctx, policyID, outcome, limit)
		case cmd.OutputFormatJSON:
			return samplesIndexedJSON(ctx, policyID, outcome, limit)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func samplesIndexedJSON(
	ctx context.Context,
	policyID string,
	outcome *string,
	limit *uint,
) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	// Build predicates for outcome filter
	var predicates []structs.QueryPredicate
	if outcome != nil {
		predicates = append(predicates, structs.QueryPredicate{
			Field: graphql.String("outcome"),
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]graphql.String{graphql.String(*outcome)},
			},
		})
	}

	records, err := searchAllEvaluationRecords(ctx, policyID, structs.SearchInput{
		First:          first,
		FullTextSearch: nil,
		Predicates:     &predicates,
		OrderBy: &structs.QueryOrder{
			Field:     "createdAt",
			Direction: "DESC",
		},
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(records)
}

func samplesIndexedTable(
	ctx context.Context,
	policyID string,
	outcome *string,
	limit *uint,
) error {
	var first *graphql.Int
	if limit != nil {
		first = graphql.NewInt(graphql.Int(*limit)) //nolint: gosec
	}

	// Build predicates for outcome filter
	var predicates []structs.QueryPredicate
	if outcome != nil {
		predicates = append(predicates, structs.QueryPredicate{
			Field: graphql.String("outcome"),
			Constraint: structs.QueryFieldConstraint{
				StringMatches: &[]graphql.String{graphql.String(*outcome)},
			},
		})
	}

	input := structs.SearchInput{
		First:          first,
		FullTextSearch: nil,
		Predicates:     &predicates,
		OrderBy: &structs.QueryOrder{
			Field:     "createdAt",
			Direction: "DESC",
		},
	}

	records, err := searchAllEvaluationRecords(ctx, policyID, input)
	if err != nil {
		return err
	}

	columns := []string{"Key", "Outcome", "Timestamp"}
	tableData := [][]string{columns}

	for _, record := range records {
		row := []string{
			record.Key,
			record.Outcome,
			cmd.HumanizeUnixSeconds(record.Timestamp),
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

func searchAllEvaluationRecords(ctx context.Context, policyID string, input structs.SearchInput) ([]evaluationRecordNode, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []evaluationRecordNode{}
	pageInput := structs.SearchInput{
		First:          graphql.NewInt(maxPageSize),
		FullTextSearch: input.FullTextSearch,
		OrderBy:        input.OrderBy,
		Predicates:     input.Predicates,
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

		result, err := searchEvaluationRecords(ctx, policyID, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Records...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = graphql.NewString(graphql.String(result.PageInfo.EndCursor))
		} else {
			break
		}
	}

	return out, nil
}

type evaluationRecordNode struct {
	Key       string `graphql:"key" json:"key"`
	Outcome   string `graphql:"outcome" json:"outcome"`
	Timestamp int    `graphql:"timestamp" json:"timestamp"`
}

type searchEvaluationRecordsResult struct {
	Records  []evaluationRecordNode
	PageInfo structs.PageInfo
}

func searchEvaluationRecords(ctx context.Context, policyID string, input structs.SearchInput) (searchEvaluationRecordsResult, error) {
	var query struct {
		Policy *struct {
			ID                      string `graphql:"id"`
			SearchEvaluationRecords struct {
				Edges []struct {
					Node evaluationRecordNode `graphql:"node"`
				} `graphql:"edges"`
				PageInfo structs.PageInfo `graphql:"pageInfo"`
			} `graphql:"searchEvaluationRecords(input: $input)"`
		} `graphql:"policy(id: $id)"`
	}

	variables := map[string]interface{}{
		"id":    graphql.ID(policyID),
		"input": input,
	}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return searchEvaluationRecordsResult{}, errors.Wrap(err, "failed to search evaluation records")
	}

	if query.Policy == nil {
		return searchEvaluationRecordsResult{}, errors.Errorf("policy with ID %q not found", policyID)
	}

	nodes := make([]evaluationRecordNode, 0)
	for _, q := range query.Policy.SearchEvaluationRecords.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchEvaluationRecordsResult{
		Records:  nodes,
		PageInfo: query.Policy.SearchEvaluationRecords.PageInfo,
	}, nil
}
