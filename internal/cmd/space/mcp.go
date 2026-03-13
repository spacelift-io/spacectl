package space

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func RegisterMCPTools(s *server.MCPServer) {
	registerListSpacesTool(s)
	registerGetSpaceTool(s)
}

type spaceNode struct {
	ID              string   `graphql:"id" json:"id"`
	Name            string   `graphql:"name" json:"name"`
	Description     string   `graphql:"description" json:"description"`
	ParentSpace     *string  `graphql:"parentSpace" json:"parentSpace"`
	InheritEntities bool     `graphql:"inheritEntities" json:"inheritEntities"`
	AccessLevel     string   `graphql:"accessLevel" json:"accessLevel"`
	Labels          []string `graphql:"labels" json:"labels"`
}

func registerListSpacesTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_spaces",
		mcp.WithDescription(`Retrieve the list of all spaces in the Spacelift account. Spaces organize resources hierarchically and control access.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Spaces",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)

		var query struct {
			Spaces []spaceNode `graphql:"spaces"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{}); err != nil {
			return nil, errors.Wrap(err, "failed to query spaces")
		}

		if len(query.Spaces) == 0 {
			return mcp.NewToolResultText("No spaces found."), nil
		}

		spacesJSON, err := json.Marshal(query.Spaces)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal spaces to JSON")
		}

		output := fmt.Sprintf("Found %d spaces:\n%s", len(query.Spaces), string(spacesJSON))
		return mcp.NewToolResultText(output), nil
	})
}

func registerGetSpaceTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_space",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift space including its hierarchy and access level.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Space",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("space_id", mcp.Description("The ID of the space to retrieve"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		spaceID, err := request.RequireString("space_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Space *spaceNode `graphql:"space(id: $spaceId)"`
		}

		variables := map[string]any{
			"spaceId": graphql.ID(spaceID),
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for space ID %q", spaceID)
		}

		if query.Space == nil {
			return mcp.NewToolResultText("Space not found"), nil
		}

		spaceJSON, err := json.Marshal(query.Space)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal space to JSON")
		}

		return mcp.NewToolResultText(string(spaceJSON)), nil
	})
}
