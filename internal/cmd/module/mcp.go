package module

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

// RegisterMCPTools registers module-related MCP tools
func RegisterMCPTools(s *server.MCPServer) {
	registerListModulesTool(s)
	registerGetModuleTool(s)
	registerListModuleVersionsTool(s)
	registerGetModuleVersionTool(s)
	registerSearchModulesTool(s)
}

func registerListModulesTool(s *server.MCPServer) {
	modulesTool := mcp.NewTool("list_modules",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift modules in the private module registry. Returns basic module information including current and latest versions.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Modules",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of modules to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on module name, description, and provider")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(modulesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		result, err := searchModulesMCP(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search modules")
		}

		modulesJSON, err := json.Marshal(result.Modules)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal modules to JSON")
		}

		output := string(modulesJSON)

		if len(result.Modules) == 0 {
			if nextPageCursor != nil {
				output = "No more modules available. You have reached the end of the list."
			} else {
				output = "No modules found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d modules:\n%s\n\nThis is not the complete list. To view more modules, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Modules), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d modules:\n%s\n\nThis is the complete list. There are no more modules to fetch.", len(result.Modules), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetModuleTool(s *server.MCPServer) {
	moduleTool := mcp.NewTool("get_module",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift module including metadata, inputs, outputs, and version information.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Module",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("module_id", mcp.Description("The ID of the module to retrieve"), mcp.Required()),
	)

	s.AddTool(moduleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		moduleID, err := request.RequireString("module_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Module *moduleDetailQuery `graphql:"module(id: $moduleId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]any{"moduleId": moduleID}); err != nil {
			return nil, errors.Wrap(err, "failed to query module")
		}

		if query.Module == nil {
			return mcp.NewToolResultText("Module not found"), nil
		}

		moduleJSON, err := json.Marshal(query.Module)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal module to JSON")
		}

		return mcp.NewToolResultText(string(moduleJSON)), nil
	})
}

func registerListModuleVersionsTool(s *server.MCPServer) {
	moduleVersionsTool := mcp.NewTool("list_module_versions",
		mcp.WithDescription(`Retrieve a list of versions for a specific Spacelift module. Shows version numbers, states, and creation dates.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Module Versions",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("module_id", mcp.Description("The ID of the module"), mcp.Required()),
		mcp.WithBoolean("include_failed", mcp.Description("Include failed versions in the results (default: false)")),
		mcp.WithNumber("limit", mcp.Description("The maximum number of versions to return, default is 50")),
	)

	s.AddTool(moduleVersionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		moduleID, err := request.RequireString("module_id")
		if err != nil {
			return nil, err
		}

		includeFailed := request.GetBool("include_failed", false)
		limit := request.GetInt("limit", 50)

		var query struct {
			Module *struct {
				Versions []moduleVersionQuery `graphql:"versions(first: $limit, includeFailed: $includeFailed)"`
			} `graphql:"module(id: $moduleId)"`
		}

		variables := map[string]any{
			"moduleId":      moduleID,
			"limit":         limit,
			"includeFailed": includeFailed,
		}

		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrap(err, "failed to query module versions")
		}

		if query.Module == nil {
			return mcp.NewToolResultText("Module not found"), nil
		}

		versionsJSON, err := json.Marshal(query.Module.Versions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal versions to JSON")
		}

		output := string(versionsJSON)
		if len(query.Module.Versions) == 0 {
			output = fmt.Sprintf("No versions found for module %s", moduleID)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetModuleVersionTool(s *server.MCPServer) {
	moduleVersionTool := mcp.NewTool("get_module_version",
		mcp.WithDescription(`Retrieve detailed information about a specific version of a Spacelift module including metadata, inputs, outputs, dependencies, and consumer information.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Module Version",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("module_id", mcp.Description("The ID of the module"), mcp.Required()),
		mcp.WithString("version_id", mcp.Description("The ID of the version to retrieve"), mcp.Required()),
	)

	s.AddTool(moduleVersionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		moduleID, err := request.RequireString("module_id")
		if err != nil {
			return nil, err
		}

		versionID, err := request.RequireString("version_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Module *struct {
				Version *moduleVersionDetailQuery `graphql:"version(id: $versionId)"`
			} `graphql:"module(id: $moduleId)"`
		}

		variables := map[string]any{
			"moduleId":  moduleID,
			"versionId": versionID,
		}

		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrap(err, "failed to query module version")
		}

		if query.Module == nil {
			return mcp.NewToolResultText("Module not found"), nil
		}

		if query.Module.Version == nil {
			return mcp.NewToolResultText("Module version not found"), nil
		}

		versionJSON, err := json.Marshal(query.Module.Version)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal version to JSON")
		}

		return mcp.NewToolResultText(string(versionJSON)), nil
	})
}

func registerSearchModulesTool(s *server.MCPServer) {
	searchModulesTool := mcp.NewTool("search_modules",
		mcp.WithDescription(`Search Spacelift modules with advanced filtering capabilities. Supports filtering by provider, labels, public/private status, and more.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search Modules",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("search", mcp.Description("Full text search query")),
		mcp.WithString("provider", mcp.Description("Filter by Terraform provider (e.g., 'aws', 'gcp', 'azure')")),
		mcp.WithString("space", mcp.Description("Filter by space ID")),
		mcp.WithArray("labels", mcp.Description("Filter by labels (array of strings)")),
		mcp.WithBoolean("public_only", mcp.Description("Only return public modules (default: false)")),
		mcp.WithNumber("limit", mcp.Description("The maximum number of modules to return, default is 50")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(searchModulesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := request.GetInt("limit", 50)

		var fullTextSearch *graphql.String
		if searchParam := request.GetString("search", ""); searchParam != "" {
			fullTextSearch = graphql.NewString(graphql.String(searchParam))
		}

		var nextPageCursor *graphql.String
		if cursor := request.GetString("next_page_cursor", ""); cursor != "" {
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		// Build predicates for advanced filtering
		var predicates []structs.QueryPredicate

		if provider := request.GetString("provider", ""); provider != "" {
			predicates = append(predicates, structs.QueryPredicate{
				Field: graphql.String("terraformProvider"),
				Constraint: structs.QueryFieldConstraint{
					StringMatches: &[]graphql.String{graphql.String(provider)},
				},
			})
		}

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

		if request.GetBool("public_only", false) {
			predicates = append(predicates, structs.QueryPredicate{
				Field: graphql.String("public"),
				Constraint: structs.QueryFieldConstraint{
					BooleanEquals: &[]graphql.Boolean{graphql.Boolean(true)},
				},
			})
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
			Predicates:     &predicates,
		}

		result, err := searchModulesMCP(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search modules")
		}

		modulesJSON, err := json.Marshal(result.Modules)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal modules to JSON")
		}

		output := string(modulesJSON)

		if len(result.Modules) == 0 {
			if nextPageCursor != nil {
				output = "No more modules available. You have reached the end of the list."
			} else {
				output = "No modules found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d modules:\n%s\n\nThis is not the complete list. To view more modules, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Modules), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d modules:\n%s\n\nThis is the complete list. There are no more modules to fetch.", len(result.Modules), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

// GraphQL query types for modules
type moduleDetailQuery struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         *string             `json:"description"`
	TerraformProvider   string              `json:"terraformProvider"`
	Repository          string              `json:"repository"`
	Branch              string              `json:"branch"`
	ProjectRoot         *string             `json:"projectRoot"`
	Public              bool                `json:"public"`
	Labels              []string            `json:"labels"`
	CreatedAt           int                 `json:"createdAt"`
	Current             *moduleVersionQuery `json:"current"`
	Latest              *moduleVersionQuery `json:"latest"`
	ModuleSource        string              `json:"moduleSource"`
	OwnerSubdomain      string              `json:"ownerSubdomain"`
	WorkflowTool        string              `json:"workflowTool"`
	LocalPreviewEnabled bool                `json:"localPreviewEnabled"`
	IsDisabled          bool                `json:"isDisabled"`
	VersionsCount       int                 `json:"versionsCount"`
	SpaceDetails        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"spaceDetails"`
}

type moduleVersionQuery struct {
	ID        string `json:"id"`
	Number    string `json:"number"`
	State     string `json:"state"`
	CreatedAt int    `json:"createdAt"`
	Notes     string `json:"notes"`
	Yanked    bool   `json:"yanked"`
}

type moduleVersionDetailQuery struct {
	ID            string                    `json:"id"`
	Number        string                    `json:"number"`
	State         string                    `json:"state"`
	CreatedAt     int                       `json:"createdAt"`
	Notes         string                    `json:"notes"`
	Yanked        bool                      `json:"yanked"`
	DownloadLink  *string                   `json:"downloadLink"`
	SourceURL     string                    `json:"sourceURL"`
	Metadata      *moduleRepositoryMetadata `json:"metadata"`
	Consumers     []moduleVersionConsumer   `json:"consumers"`
	ConsumerCount int                       `json:"consumerCount"`
	Commit        struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		URL     string `json:"url"`
	} `json:"commit"`
}

type moduleRepositoryMetadata struct {
	Root *moduleMetadata `json:"root"`
}

type moduleMetadata struct {
	Name                 string                     `json:"name"`
	Path                 string                     `json:"path"`
	Readme               string                     `json:"readme"`
	Changelog            string                     `json:"changelog"`
	Empty                bool                       `json:"empty"`
	Inputs               []moduleInput              `json:"inputs"`
	Outputs              []moduleOutput             `json:"outputs"`
	Dependencies         []string                   `json:"dependencies"`
	ProviderDependencies []moduleProviderDependency `json:"providerDependencies"`
	Resources            []moduleResource           `json:"resources"`
}

type moduleInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Type        string  `json:"type"`
	Default     *string `json:"default"`
	Required    bool    `json:"required"`
}

type moduleOutput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type moduleProviderDependency struct {
	Name               string `json:"name"`
	Source             string `json:"source"`
	VersionConstraints string `json:"versionConstraints"`
}

type moduleResource struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type moduleVersionConsumer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mcpSearchModulesResult struct {
	Modules  []module
	PageInfo structs.PageInfo
}

// searchModulesMCP executes a search query for modules for MCP tools
func searchModulesMCP(ctx context.Context, input structs.SearchInput) (*mcpSearchModulesResult, error) {
	var query struct {
		SearchModulesOutput struct {
			Edges []struct {
				Node module `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchModules(input: $input)"`
	}

	variables := map[string]any{"input": input}

	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to execute modules search query")
	}

	nodes := make([]module, 0)
	for _, q := range query.SearchModulesOutput.Edges {
		nodes = append(nodes, q.Node)
	}

	return &mcpSearchModulesResult{
		Modules:  nodes,
		PageInfo: query.SearchModulesOutput.PageInfo,
	}, nil
}
