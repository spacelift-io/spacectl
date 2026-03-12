package apikey

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
	registerListAPIKeysTool(s)
	registerGetAPIKeyTool(s)
}

type apiKeyNode struct {
	ID          string   `graphql:"id" json:"id"`
	Name        string   `graphql:"name" json:"name"`
	Description string   `graphql:"description" json:"description"`
	Type        string   `graphql:"type" json:"type"`
	Admin       bool     `graphql:"admin" json:"admin"`
	Creator     string   `graphql:"creator" json:"creator"`
	CreatedAt   int      `graphql:"createdAt" json:"createdAt"`
	LastUsedAt  *int     `graphql:"lastUsedAt" json:"lastUsedAt"`
	Deleted     bool     `graphql:"deleted" json:"deleted"`
	Teams       []string `graphql:"teams" json:"teams"`
	TeamCount   int      `graphql:"teamCount" json:"teamCount"`
	SpaceCount  int      `graphql:"spaceCount" json:"spaceCount"`
	IsMachineUser bool   `graphql:"isMachineUser" json:"isMachineUser"`
	ExpiresAt   *int     `graphql:"expiresAt" json:"expiresAt"`
}

type apiKeyDetail struct {
	ID          string   `graphql:"id" json:"id"`
	Name        string   `graphql:"name" json:"name"`
	Description string   `graphql:"description" json:"description"`
	Type        string   `graphql:"type" json:"type"`
	Admin       bool     `graphql:"admin" json:"admin"`
	Creator     string   `graphql:"creator" json:"creator"`
	CreatedAt   int      `graphql:"createdAt" json:"createdAt"`
	LastUsedAt  *int     `graphql:"lastUsedAt" json:"lastUsedAt"`
	Deleted     bool     `graphql:"deleted" json:"deleted"`
	Teams       []string `graphql:"teams" json:"teams"`
	TeamCount   int      `graphql:"teamCount" json:"teamCount"`
	SpaceCount  int      `graphql:"spaceCount" json:"spaceCount"`
	IsMachineUser bool   `graphql:"isMachineUser" json:"isMachineUser"`
	ExpiresAt   *int     `graphql:"expiresAt" json:"expiresAt"`
	AccessRules []apiKeyAccessRule `graphql:"accessRules" json:"accessRules"`
}

type apiKeyAccessRule struct {
	Space         string `graphql:"space" json:"space"`
	SpaceAccessLevel string `graphql:"spaceAccessLevel" json:"spaceAccessLevel"`
}

func registerListAPIKeysTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_api_keys",
		mcp.WithDescription(`Retrieve the list of all API keys for the Spacelift account. Returns key metadata including name, type, creator, and usage information. Does not return the secret values.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List API Keys",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)

		var query struct {
			APIKeys []apiKeyNode `graphql:"apiKeys"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{}); err != nil {
			return nil, errors.Wrap(err, "failed to query API keys")
		}

		if len(query.APIKeys) == 0 {
			return mcp.NewToolResultText("No API keys found."), nil
		}

		keysJSON, err := json.Marshal(query.APIKeys)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal API keys to JSON")
		}

		output := fmt.Sprintf("Found %d API keys:\n%s", len(query.APIKeys), string(keysJSON))
		return mcp.NewToolResultText(output), nil
	})
}

func registerGetAPIKeyTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_api_key",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift API key including its access rules and space permissions.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get API Key",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("api_key_id", mcp.Description("The ID of the API key to retrieve"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		apiKeyID, err := request.RequireString("api_key_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			APIKey *apiKeyDetail `graphql:"apiKey(id: $id)"`
		}

		variables := map[string]any{
			"id": graphql.ID(apiKeyID),
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for API key ID %q", apiKeyID)
		}

		if query.APIKey == nil {
			return mcp.NewToolResultText("API key not found"), nil
		}

		keyJSON, err := json.Marshal(query.APIKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal API key to JSON")
		}

		return mcp.NewToolResultText(string(keyJSON)), nil
	})
}
