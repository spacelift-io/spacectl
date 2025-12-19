package module

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// RegisterMCPTools registers module-related MCP tools
func RegisterMCPTools(s *server.MCPServer) {
	registerModuleGuideTool(s)
	registerListModulesTool(s)
	registerGetModuleTool(s)
	registerListModuleVersionsTool(s)
	registerGetModuleVersionTool(s)
	registerSearchModulesTool(s)
}

// registerModuleGuideTool registers the module operations guide tool
func registerModuleGuideTool(s *server.MCPServer) {
	moduleGuideTool := mcp.NewTool("get_module_guide",
		mcp.WithDescription(`Get comprehensive guidance on working with Spacelift modules through MCP tools, including parameter explanations, common patterns, and troubleshooting tips.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Module Operations Guide",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("topic", mcp.Description("Specific topic to focus on: 'all', 'search', 'filtering', 'versioning', 'terraform_providers'"),
			mcp.DefaultString("all"),
			mcp.Enum("all", "search", "filtering", "versioning", "terraform_providers")),
	)

	s.AddTool(moduleGuideTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		topic := request.GetString("topic", "all")

		var guide strings.Builder
		guide.WriteString("Spacelift Module Operations Guide\n")
		guide.WriteString("==================================\n\n")

		guide.WriteString("Available MCP Tools for Module Management:\n")
		guide.WriteString("- list_modules: Browse all modules with pagination\n")
		guide.WriteString("- search_modules: Advanced search with filtering\n")
		guide.WriteString("- get_module: Get detailed module information\n")
		guide.WriteString("- list_module_versions: List versions for a module\n")
		guide.WriteString("- get_module_version: Get detailed version information\n\n")

		if topic == "all" || topic == "search" {
			guide.WriteString("SEARCH AND DISCOVERY\n")
			guide.WriteString("====================\n\n")
			guide.WriteString("Basic Search (list_modules):\n")
			guide.WriteString("- Use for simple text search across module names and descriptions\n")
			guide.WriteString("- Parameters: search (text), limit (number), next_page_cursor (pagination)\n")
			guide.WriteString("- Example: list_modules with search=\"worker pool\"\n\n")

			guide.WriteString("Advanced Search (search_modules):\n")
			guide.WriteString("- Use for complex filtering with multiple criteria\n")
			guide.WriteString("- Supports filtering by terraform provider, space, labels, visibility\n")
			guide.WriteString("- Parameters: search, terraform_provider, space, labels, public_only, limit, next_page_cursor\n")
			guide.WriteString("- Example: search_modules with terraform_provider=\"aws\" and search=\"ec2\"\n\n")
		}

		if topic == "all" || topic == "terraform_providers" {
			guide.WriteString("TERRAFORM PROVIDER FILTERING\n")
			guide.WriteString("=============================\n\n")
			guide.WriteString("IMPORTANT: The 'terraform_provider' parameter filters by TERRAFORM providers, not VCS providers!\n\n")
			guide.WriteString("Terraform Provider Examples:\n")
			guide.WriteString("- terraform_provider=\"aws\" - for AWS resources\n")
			guide.WriteString("- terraform_provider=\"gcp\" or terraform_provider=\"google\" - for Google Cloud\n")
			guide.WriteString("- terraform_provider=\"azurerm\" - for Azure resources\n")
			guide.WriteString("- terraform_provider=\"kubernetes\" - for Kubernetes resources\n")
			guide.WriteString("- terraform_provider=\"datadog\" - for Datadog resources\n\n")

			guide.WriteString("Common Mistakes to Avoid:\n")
			guide.WriteString("❌ Don't use VCS provider values like 'GITHUB', 'GITLAB' in the terraform_provider parameter\n")
			guide.WriteString("❌ Don't use provider namespace format like 'hashicorp/aws'\n")
			guide.WriteString("✅ Use simple terraform provider names like 'aws', 'gcp', 'azurerm'\n\n")

			guide.WriteString("The provider field in module search results shows VCS provider (where code is hosted):\n")
			guide.WriteString("- \"provider\": \"GITHUB\" means the module code is on GitHub\n")
			guide.WriteString("- \"terraformProvider\": \"aws\" means the module uses AWS terraform provider\n\n")
		}

		if topic == "all" || topic == "filtering" {
			guide.WriteString("FILTERING OPTIONS\n")
			guide.WriteString("=================\n\n")
			guide.WriteString("Available Filters in search_modules:\n\n")

			guide.WriteString("1. terraform_provider (string):\n")
			guide.WriteString("   - Filters by Terraform provider (aws, gcp, azurerm, etc.)\n")
			guide.WriteString("   - Example: terraform_provider=\"aws\"\n\n")

			guide.WriteString("2. space (string):\n")
			guide.WriteString("   - Filters by Spacelift space ID\n")
			guide.WriteString("   - Example: space=\"legacy\"\n\n")

			guide.WriteString("3. labels (array):\n")
			guide.WriteString("   - Filters by module labels\n")
			guide.WriteString("   - Example: labels=[\"production\", \"database\"]\n\n")

			guide.WriteString("4. public_only (boolean):\n")
			guide.WriteString("   - When true, only returns public modules\n")
			guide.WriteString("   - Example: public_only=true\n\n")

			guide.WriteString("5. search (string):\n")
			guide.WriteString("   - Full-text search across name, description\n")
			guide.WriteString("   - Example: search=\"worker pool ec2\"\n\n")
		}

		if topic == "all" || topic == "versioning" {
			guide.WriteString("MODULE VERSIONING\n")
			guide.WriteString("=================\n\n")
			guide.WriteString("Module Version Information:\n")
			guide.WriteString("- current: The currently active/recommended version\n")
			guide.WriteString("- latest: The most recently created version\n")
			guide.WriteString("- State can be: ACTIVE, PENDING, FAILED\n")
			guide.WriteString("- Yanked versions are deprecated and should not be used\n\n")

			guide.WriteString("Using Modules in Terraform:\n")
			guide.WriteString("module \"example\" {\n")
			guide.WriteString("  source  = \"app.spacelift.io/your-org/module-name/provider\"\n")
			guide.WriteString("  version = \"1.0.0\"\n")
			guide.WriteString("  \n")
			guide.WriteString("  # Module inputs here\n")
			guide.WriteString("}\n\n")

			guide.WriteString("Version States:\n")
			guide.WriteString("- ACTIVE: Ready to use, stable version\n")
			guide.WriteString("- PENDING: Being processed, not ready yet\n")
			guide.WriteString("- FAILED: Processing failed, should not be used\n\n")
		}

		guide.WriteString("TROUBLESHOOTING\n")
		guide.WriteString("===============\n\n")
		guide.WriteString("Common Issues:\n\n")

		guide.WriteString("1. \"No modules found\" with terraform_provider filter:\n")
		guide.WriteString("   - Check if the terraform provider name is correct\n")
		guide.WriteString("   - Try without terraform_provider filter first to see available modules\n")
		guide.WriteString("   - Check terraformProvider field in results\n\n")

		guide.WriteString("2. Module not showing expected results:\n")
		guide.WriteString("   - Check space access permissions\n")
		guide.WriteString("   - Verify module is not disabled (isDisabled: false)\n")
		guide.WriteString("   - Check if module is public/private as expected\n\n")

		guide.WriteString("Best Practices:\n")
		guide.WriteString("- Always check current.state before using a module version\n")
		guide.WriteString("- Use specific version numbers in Terraform, not 'latest'\n")
		guide.WriteString("- Review module inputs/outputs before implementation\n")
		guide.WriteString("- Check module documentation and examples\n")

		return mcp.NewToolResultText(guide.String()), nil
	})
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
		authenticated.Ensure(ctx, nil)
		moduleID, err := request.RequireString("module_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			Module *moduleDetailQuery `graphql:"module(id: $moduleId)"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{"moduleId": moduleID}); err != nil {
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
		authenticated.Ensure(ctx, nil)
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

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
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

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
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
		mcp.WithString("terraform_provider", mcp.Description("Filter by Terraform provider (e.g., 'aws', 'gcp', 'azure')")),
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

		if provider := request.GetString("terraform_provider", ""); provider != "" {
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
	SourceURL     string                    `json:"sourceURL" graphql:"sourceURL"`
	Metadata      *moduleRepositoryMetadata `json:"metadata"`
	Consumers     []moduleVersionConsumer   `json:"consumers"`
	ConsumerCount int                       `json:"consumerCount"`
	Commit        struct {
		Hash        string  `json:"hash"`
		Message     string  `json:"message"`
		AuthorName  string  `json:"authorName"`
		AuthorLogin *string `json:"authorLogin"`
		URL         string  `json:"url"`
		Timestamp   int     `json:"timestamp"`
		Tag         string  `json:"tag"`
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
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

type moduleOutput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type moduleProviderDependency struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	Version   string `json:"version"`
}

type moduleResource struct {
	Type string `json:"type"`
	Name string `json:"name"`
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

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to execute modules search query")
	}

	if len(query.SearchModulesOutput.Edges) == 0 {
		_, err := authenticated.CurrentViewer(ctx)
		if errors.Is(err, authenticated.ErrViewerUnknown) {
			return nil, errors.New("You are not logged in, could not find modules")
		}
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
