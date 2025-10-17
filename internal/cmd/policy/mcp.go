package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// RegisterMCPTools registers policy-related MCP tools
func RegisterMCPTools(s *server.MCPServer) {
	registerListPoliciesTool(s)
	registerGetPolicyTool(s)
	registerListPolicySamplesTool(s)
	registerGetPolicySampleTool(s)
	registerListPolicySamplesIndexedTool(s)
}

func registerListPoliciesTool(s *server.MCPServer) {
	policiesTool := mcp.NewTool("list_policies",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift policies. Returns policies you have access to with their metadata.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Policies",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of policies to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on policy name and description")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(policiesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := request.GetInt("limit", 50)

		var fullTextSearch *graphql.String
		if searchParam := request.GetString("search", ""); searchParam != "" {
			fullTextSearch = graphql.NewString(graphql.String(searchParam))
		}

		var nextPageCursor *graphql.String
		if cursor := request.GetString("next_page_cursor", ""); cursor != "" {
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
		}

		result, err := searchPolicies(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search policies")
		}

		policiesJSON, err := json.Marshal(result.Policies)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal policies to JSON")
		}

		output := string(policiesJSON)

		if len(result.Policies) == 0 {
			if nextPageCursor != nil {
				output = "No more policies available. You have reached the end of the list."
			} else {
				output = "No policies found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d policies:\n%s\n\nThis is not the complete list. To view more policies, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Policies), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d policies:\n%s\n\nThis is the complete list. There are no more policies to fetch.", len(result.Policies), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetPolicyTool(s *server.MCPServer) {
	policyTool := mcp.NewTool("get_policy",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift policy including its body, type, and metadata.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Policy",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("policy_id", mcp.Description("The ID of the policy to retrieve"), mcp.Required()),
	)

	s.AddTool(policyTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		policyID, err := request.RequireString("policy_id")
		if err != nil {
			return nil, err
		}

		policy, found, err := getPolicyByID(ctx, policyID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query for policy ID %q", policyID)
		}

		if !found {
			return mcp.NewToolResultText("Policy not found"), nil
		}

		policyJSON, err := json.Marshal(policy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal policy to JSON")
		}

		return mcp.NewToolResultText(string(policyJSON)), nil
	})
}

func registerListPolicySamplesTool(s *server.MCPServer) {
	samplesTool := mcp.NewTool("list_policy_samples",
		mcp.WithDescription(`Retrieve a list of evaluation samples for a specific Spacelift policy. Shows sample keys, outcomes, and timestamps.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Policy Samples",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("policy_id", mcp.Description("The ID of the policy"), mcp.Required()),
	)

	s.AddTool(samplesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		policyID, err := request.RequireString("policy_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Policy *policyEvaluation `graphql:"policy(id: $policyId)"`
		}

		variables := map[string]interface{}{
			"policyId": graphql.ID(policyID),
		}

		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for policy ID %q", policyID)
		}

		if query.Policy == nil {
			return mcp.NewToolResultText("Policy not found"), nil
		}

		samplesJSON, err := json.Marshal(query.Policy.EvaluationRecords)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal samples to JSON")
		}

		output := string(samplesJSON)
		if len(query.Policy.EvaluationRecords) == 0 {
			output = fmt.Sprintf("No evaluation samples found for policy %s", policyID)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetPolicySampleTool(s *server.MCPServer) {
	sampleTool := mcp.NewTool("get_policy_sample",
		mcp.WithDescription(`Retrieve a specific evaluation sample for a Spacelift policy by key. Returns the sample input and policy body.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Policy Sample",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("policy_id", mcp.Description("The ID of the policy"), mcp.Required()),
		mcp.WithString("sample_key", mcp.Description("The key of the sample to retrieve"), mcp.Required()),
	)

	s.AddTool(sampleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		policyID, err := request.RequireString("policy_id")
		if err != nil {
			return nil, err
		}

		sampleKey, err := request.RequireString("sample_key")
		if err != nil {
			return nil, err
		}

		var query struct {
			Policy *struct {
				Sample policyEvaluationSample `graphql:"evaluationSample(key: $key)"`
			} `graphql:"policy(id: $policyId)"`
		}

		variables := map[string]interface{}{
			"policyId": graphql.ID(policyID),
			"key":      graphql.String(sampleKey),
		}

		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for policy sample")
		}

		if query.Policy == nil {
			return mcp.NewToolResultText("Policy not found"), nil
		}

		sampleJSON, err := json.Marshal(query.Policy.Sample)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal sample to JSON")
		}

		return mcp.NewToolResultText(string(sampleJSON)), nil
	})
}

func registerListPolicySamplesIndexedTool(s *server.MCPServer) {
	samplesIndexedTool := mcp.NewTool("list_policy_samples_indexed",
		mcp.WithDescription(`List indexed policy evaluation samples with pagination and full-text search support. Returns searchable evaluation records with keys, outcomes, and timestamps. Supports filtering by outcome.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Policy Samples Indexed",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("policy_id", mcp.Description("The ID of the policy"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("The maximum number of samples to return, default is 50")),
		mcp.WithString("search", mcp.Description("Full text search query for filtering evaluation records")),
		mcp.WithString("outcome", mcp.Description("Filter by outcome (e.g., 'allow', 'deny', 'undecided')")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(samplesIndexedTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		policyID, err := request.RequireString("policy_id")
		if err != nil {
			return nil, err
		}

		limit := request.GetInt("limit", 50)

		var fullTextSearch *graphql.String
		if searchParam := request.GetString("search", ""); searchParam != "" {
			fullTextSearch = graphql.NewString(graphql.String(searchParam))
		}

		var nextPageCursor *graphql.String
		if cursor := request.GetString("next_page_cursor", ""); cursor != "" {
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		// Build predicates for outcome filter
		var predicates []structs.QueryPredicate
		if outcomeParam := request.GetString("outcome", ""); outcomeParam != "" {
			predicates = append(predicates, structs.QueryPredicate{
				Field: graphql.String("outcome"),
				Constraint: structs.QueryFieldConstraint{
					StringMatches: &[]graphql.String{graphql.String(outcomeParam)},
				},
			})
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
			Predicates:     &predicates,
			OrderBy: &structs.QueryOrder{
				Field:     "createdAt",
				Direction: "DESC",
			},
		}

		result, err := searchEvaluationRecords(ctx, policyID, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search evaluation records")
		}

		recordsJSON, err := json.Marshal(result.Records)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal evaluation records to JSON")
		}

		output := string(recordsJSON)

		if len(result.Records) == 0 {
			if nextPageCursor != nil {
				output = fmt.Sprintf("No more evaluation records available for policy %s. You have reached the end of the list.", policyID)
			} else {
				output = fmt.Sprintf("No evaluation records found for policy %s matching your criteria.", policyID)
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d evaluation records for policy %s:\n%s\n\nThis is not the complete list. To view more records, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Records), policyID, output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d evaluation records for policy %s:\n%s\n\nThis is the complete list. There are no more records to fetch.", len(result.Records), policyID, output)
		}

		return mcp.NewToolResultText(output), nil
	})
}
