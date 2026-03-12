package blueprint

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
	registerListBlueprintsTool(s)
	registerGetBlueprintTool(s)
}

func registerListBlueprintsTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_blueprints",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift blueprints. Blueprints are reusable templates for creating stacks.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Blueprints",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of blueprints to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on blueprint name and description")),
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

		result, err := searchBlueprints(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search blueprints")
		}

		blueprintsJSON, err := json.Marshal(result.Blueprints)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal blueprints to JSON")
		}

		output := string(blueprintsJSON)

		if len(result.Blueprints) == 0 {
			if nextPageCursor != nil {
				output = "No more blueprints available. You have reached the end of the list."
			} else {
				output = "No blueprints found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d blueprints:\n%s\n\nThis is not the complete list. To view more blueprints, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Blueprints), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d blueprints:\n%s\n\nThis is the complete list. There are no more blueprints to fetch.", len(result.Blueprints), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetBlueprintTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_blueprint",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift blueprint including its template, inputs, and metadata.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Blueprint",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("blueprint_id", mcp.Description("The ID of the blueprint to retrieve"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		blueprintID, err := request.RequireString("blueprint_id")
		if err != nil {
			return nil, err
		}

		b, found, err := getBlueprintByID(ctx, blueprintID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query for blueprint ID %q", blueprintID)
		}

		if !found {
			return mcp.NewToolResultText("Blueprint not found"), nil
		}

		blueprintJSON, err := json.Marshal(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal blueprint to JSON")
		}

		return mcp.NewToolResultText(string(blueprintJSON)), nil
	})
}
