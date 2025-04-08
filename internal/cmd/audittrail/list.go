package audittrail

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var defaultOrder = structs.QueryOrder{
	Field:     "createdAt",
	Direction: "DESC",
}

func listAuditTrails() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		outputFormat, err := internalCmd.GetOutputFormat(cmd)
		if err != nil {
			return err
		}

		var limit *uint
		if cmd.IsSet(internalCmd.FlagLimit.Name) {
			if cmd.Uint(internalCmd.FlagLimit.Name) >= math.MaxInt32 {
				return fmt.Errorf("limit must be less than %d", math.MaxInt32)
			}

			limit = internal.Ptr(uint(cmd.Uint(internalCmd.FlagLimit.Name)))
		}

		var search *string
		if cmd.IsSet(internalCmd.FlagSearch.Name) {
			if cmd.String(internalCmd.FlagSearch.Name) == "" {
				return fmt.Errorf("search must be non-empty")
			}

			search = internal.Ptr(cmd.String(internalCmd.FlagSearch.Name))
		}

		switch outputFormat {
		case internalCmd.OutputFormatTable:
			err := listAuditTrailEntriesTable(ctx, search, limit)
			return err
		case internalCmd.OutputFormatJSON:
			err := listAuditTrailEntriesJSON(ctx, search, limit)
			return err
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listAuditTrailEntriesTable(
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

	input := structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
		OrderBy:        &defaultOrder,
	}

	entries, err := searchAllAuditTrailEntries(ctx, input)
	if err != nil {
		return err
	}

	columns := []string{"Action", "Type", "Affected Resource", "Related Resource", "Created By", "Created At"}

	tableData := [][]string{columns}
	for _, b := range entries {
		row := []string{
			b.Action,
			b.EventType,
			formatAuditTrailResource(&b.AffectedResource),
			formatAuditTrailResource(b.RelatedResource),
			b.Actor.Username,
			internalCmd.HumanizeUnixSeconds(b.CreatedAt),
		}

		tableData = append(tableData, row)
	}

	return internalCmd.OutputTable(tableData, true)
}

func listAuditTrailEntriesJSON(
	ctx context.Context,
	search *string,
	limit *uint,
) error {
	var first *graphql.Int
	if limit != nil {
		//nolint: gosec
		first = graphql.NewInt(graphql.Int(*limit))
	}

	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = graphql.NewString(graphql.String(*search))
	}

	auditTrailEntries, err := searchAllAuditTrailEntries(ctx, structs.SearchInput{
		First:          first,
		FullTextSearch: fullTextSearch,
		OrderBy:        &defaultOrder,
	})
	if err != nil {
		return err
	}

	return internalCmd.OutputJSON(auditTrailEntries)
}

func searchAllAuditTrailEntries(ctx context.Context, input structs.SearchInput) ([]auditTrailEntryNode, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []auditTrailEntryNode{}
	pageInput := structs.SearchInput{
		First:          graphql.NewInt(maxPageSize),
		FullTextSearch: input.FullTextSearch,
		OrderBy:        input.OrderBy,
	}
	for {
		if !fetchAll {
			pageInput.First = graphql.NewInt(
				//nolint: gosec
				graphql.Int(
					slices.Min([]int{maxPageSize, limit - len(out)}),
				),
			)
		}

		result, err := searchAuditTrailEntries(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.AuditTrailEntries...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = graphql.NewString(graphql.String(result.PageInfo.EndCursor))
		} else {
			break
		}
	}

	return out, nil
}

func searchAuditTrailEntries(ctx context.Context, input structs.SearchInput) (searchAuditTrailEntriesResult, error) {
	var query struct {
		SearchAuditTrailEntriesOutput struct {
			Edges []struct {
				Node auditTrailEntryNode `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchAuditTrailEntries(input: $input)"`
	}

	if err := authenticated.Client.Query(
		ctx,
		&query,
		map[string]interface{}{"input": input},
	); err != nil {
		return searchAuditTrailEntriesResult{}, errors.Wrap(err, "failed search for audit trail entries")
	}

	nodes := make([]auditTrailEntryNode, 0)
	for _, q := range query.SearchAuditTrailEntriesOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return searchAuditTrailEntriesResult{
		AuditTrailEntries: nodes,
		PageInfo:          query.SearchAuditTrailEntriesOutput.PageInfo,
	}, nil
}

func formatAuditTrailResource(resource *auditTrailResource) string {
	if resource == nil {
		return ""
	}

	formatted := internalCmd.HumanizeAuditTrailResourceType(resource.ResourceType)

	if resource.ResourceID != nil {
		formatted += " - " + *resource.ResourceID
	}

	return formatted
}

type searchAuditTrailEntriesResult struct {
	AuditTrailEntries []auditTrailEntryNode
	PageInfo          structs.PageInfo
}

type auditTrailResource struct {
	ResourceID   *string `json:"resourceId" graphql:"resourceId"`
	ResourceType string  `json:"resourceType" graphql:"resourceType"`
}

type auditTrailEntryNode struct {
	ID     string `json:"id" graphql:"id"`
	Action string `json:"action" graphql:"action"`
	Actor  struct {
		Run *struct {
			ID      string `json:"id" graphql:"id"`
			StackID string `json:"stackId" graphql:"stackId"`
		} `json:"run" graphql:"run"`
		Username string `json:"username" graphql:"username"`
	} `json:"actor" graphql:"actor"`
	AffectedResource auditTrailResource  `json:"affectedResource" graphql:"affectedResource"`
	Body             *string             `json:"body" graphql:"body"`
	EventType        string              `json:"eventType" graphql:"eventType"`
	RelatedResource  *auditTrailResource `json:"relatedResource" graphql:"relatedResource"`
	CreatedAt        int                 `json:"createdAt" graphql:"createdAt"`
	UpdatedAt        int                 `json:"updatedAt" graphql:"updatedAt"`
}
