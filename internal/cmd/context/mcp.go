package context

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

func RegisterMCPTools(s *server.MCPServer) {
	registerListContextsTool(s)
	registerGetContextTool(s)
	registerSearchContextsTool(s)
}

type contextNode struct {
	ID        string   `graphql:"id" json:"id"`
	Name      string   `graphql:"name" json:"name"`
	Labels    []string `graphql:"labels" json:"labels"`
	CreatedAt int      `graphql:"createdAt" json:"createdAt"`
	UpdatedAt int      `graphql:"updatedAt" json:"updatedAt"`
	Space     struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	Description    *string `graphql:"description" json:"description"`
	HasAttachedStacks bool `graphql:"hasAttachedStacks" json:"hasAttachedStacks"`
}

type contextDetail struct {
	ID        string   `graphql:"id" json:"id"`
	Name      string   `graphql:"name" json:"name"`
	Labels    []string `graphql:"labels" json:"labels"`
	CreatedAt int      `graphql:"createdAt" json:"createdAt"`
	UpdatedAt int      `graphql:"updatedAt" json:"updatedAt"`
	Space     struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	Description    *string         `graphql:"description" json:"description"`
	HasAttachedStacks bool         `graphql:"hasAttachedStacks" json:"hasAttachedStacks"`
	Config         []configElement `graphql:"config" json:"config"`
	AttachedStacks []contextStackAttachment `graphql:"attachedStacks" json:"attachedStacks"`
}

type configElement struct {
	ID          string  `graphql:"id" json:"id"`
	Type        string  `graphql:"type" json:"type"`
	Value       *string `graphql:"value" json:"value"`
	WriteOnly   bool    `graphql:"writeOnly" json:"writeOnly"`
	Description string  `graphql:"description" json:"description"`
	Checksum    string  `graphql:"checksum" json:"checksum"`
	Runtime     bool    `graphql:"runtime" json:"runtime"`
}

type contextStackAttachment struct {
	StackID        string `graphql:"stackId" json:"stackId"`
	StackName      string `graphql:"stackName" json:"stackName"`
	IsAutoattached bool   `graphql:"isAutoattached" json:"isAutoattached"`
	Priority       int    `graphql:"priority" json:"priority"`
}

type searchContextsResult struct {
	Contexts []contextNode
	PageInfo structs.PageInfo
}

func searchContexts(ctx context.Context, input structs.SearchInput) (*searchContextsResult, error) {
	var query struct {
		SearchContextsOutput struct {
			Edges []struct {
				Node contextNode `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchContexts(input: $input)"`
	}

	variables := map[string]any{"input": input}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to execute contexts search query")
	}

	nodes := make([]contextNode, 0, len(query.SearchContextsOutput.Edges))
	for _, edge := range query.SearchContextsOutput.Edges {
		nodes = append(nodes, edge.Node)
	}

	return &searchContextsResult{
		Contexts: nodes,
		PageInfo: query.SearchContextsOutput.PageInfo,
	}, nil
}

func registerListContextsTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_contexts",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift contexts. Contexts are collections of environment variables and mounted files that can be attached to stacks.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Contexts",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of contexts to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on context name, description, and labels")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
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

		result, err := searchContexts(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search contexts")
		}

		contextsJSON, err := json.Marshal(result.Contexts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal contexts to JSON")
		}

		output := string(contextsJSON)

		if len(result.Contexts) == 0 {
			if nextPageCursor != nil {
				output = "No more contexts available. You have reached the end of the list."
			} else {
				output = "No contexts found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d contexts:\n%s\n\nThis is not the complete list. To view more contexts, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Contexts), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d contexts:\n%s\n\nThis is the complete list. There are no more contexts to fetch.", len(result.Contexts), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetContextTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_context",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift context including its configuration elements (environment variables and mounted files) and attached stacks.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Context",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("context_id", mcp.Description("The ID of the context to retrieve"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		contextID, err := request.RequireString("context_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Context *contextDetail `graphql:"context(id: $contextId)"`
		}

		variables := map[string]any{
			"contextId": graphql.ID(contextID),
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for context ID %q", contextID)
		}

		if query.Context == nil {
			return mcp.NewToolResultText("Context not found"), nil
		}

		contextJSON, err := json.Marshal(query.Context)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal context to JSON")
		}

		return mcp.NewToolResultText(string(contextJSON)), nil
	})
}

func registerSearchContextsTool(s *server.MCPServer) {
	tool := mcp.NewTool("search_contexts",
		mcp.WithDescription(`Search Spacelift contexts with advanced filtering. Supports filtering by space and labels.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search Contexts",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("search", mcp.Description("Full text search query")),
		mcp.WithString("space", mcp.Description("Filter by space ID")),
		mcp.WithArray("labels", mcp.Description("Filter by labels (array of strings)")),
		mcp.WithNumber("limit", mcp.Description("The maximum number of contexts to return, default is 50")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		limit := request.GetInt("limit", 50)

		var fullTextSearch *graphql.String
		if searchParam := request.GetString("search", ""); searchParam != "" {
			fullTextSearch = graphql.NewString(graphql.String(searchParam))
		}

		var nextPageCursor *graphql.String
		if cursor := request.GetString("next_page_cursor", ""); cursor != "" {
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		var predicates []structs.QueryPredicate

		if space := request.GetString("space", ""); space != "" {
			predicates = append(predicates, structs.QueryPredicate{
				Field: graphql.String("space"),
				Constraint: structs.QueryFieldConstraint{
					StringMatches: &[]graphql.String{graphql.String(space)},
				},
			})
		}

		if labels := request.GetStringSlice("labels", nil); len(labels) > 0 {
			var labelStrings []graphql.String
			for _, label := range labels {
				labelStrings = append(labelStrings, graphql.String(label))
			}
			predicates = append(predicates, structs.QueryPredicate{
				Field: graphql.String("labels"),
				Constraint: structs.QueryFieldConstraint{
					StringMatches: &labelStrings,
				},
			})
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
			Predicates:     &predicates,
		}

		result, err := searchContexts(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search contexts")
		}

		contextsJSON, err := json.Marshal(result.Contexts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal contexts to JSON")
		}

		output := string(contextsJSON)

		if len(result.Contexts) == 0 {
			if nextPageCursor != nil {
				output = "No more contexts available. You have reached the end of the list."
			} else {
				output = "No contexts found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d contexts:\n%s\n\nThis is not the complete list. To view more contexts, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Contexts), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d contexts:\n%s\n\nThis is the complete list. There are no more contexts to fetch.", len(result.Contexts), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}
