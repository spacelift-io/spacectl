package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// RegisterMCPTools registers GraphQL schema introspection and exploration tools
func RegisterMCPTools(s *server.MCPServer) {
	registerIntrospectSchemaTool(s)
	registerGetTypeDetailsTool(s)
	registerSearchSchemaFieldsTool(s)
	registerAuthenticationGuideTool(s)
}

// registerIntrospectSchemaTool registers the schema introspection tool
func registerIntrospectSchemaTool(s *server.MCPServer) {
	introspectTool := mcp.NewTool("introspect_graphql_schema",
		mcp.WithDescription(`Introspect the complete GraphQL schema for the Spacelift API. Returns all available types, queries, mutations, and subscriptions with their fields and descriptions.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Introspect GraphQL Schema",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("format", mcp.Description("Output format: 'summary' for high-level overview, 'detailed' for complete schema"),
			mcp.DefaultString("summary"),
			mcp.Enum("summary", "detailed")),
	)

	s.AddTool(introspectTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		format := request.GetString("format", "summary")

		var query struct {
			Schema struct {
				QueryType struct {
					Name   string `json:"name"`
					Fields []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name   string `json:"name"`
							Kind   string `json:"kind"`
							OfType *struct {
								Name string `json:"name"`
								Kind string `json:"kind"`
							} `json:"ofType"`
						} `json:"type"`
						Args []struct {
							Name        string `json:"name"`
							Description string `json:"description"`
							Type        struct {
								Name   string `json:"name"`
								Kind   string `json:"kind"`
								OfType *struct {
									Name string `json:"name"`
									Kind string `json:"kind"`
								} `json:"ofType"`
							} `json:"type"`
						} `json:"args"`
					} `json:"fields"`
				} `json:"queryType"`
				MutationType *struct {
					Name   string `json:"name"`
					Fields []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name   string `json:"name"`
							Kind   string `json:"kind"`
							OfType *struct {
								Name string `json:"name"`
								Kind string `json:"kind"`
							} `json:"ofType"`
						} `json:"type"`
						Args []struct {
							Name        string `json:"name"`
							Description string `json:"description"`
							Type        struct {
								Name   string `json:"name"`
								Kind   string `json:"kind"`
								OfType *struct {
									Name string `json:"name"`
									Kind string `json:"kind"`
								} `json:"ofType"`
							} `json:"type"`
						} `json:"args"`
					} `json:"fields"`
				} `json:"mutationType"`
				Types []struct {
					Name        string `json:"name"`
					Kind        string `json:"kind"`
					Description string `json:"description"`
					Fields      []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name   string `json:"name"`
							Kind   string `json:"kind"`
							OfType *struct {
								Name string `json:"name"`
								Kind string `json:"kind"`
							} `json:"ofType"`
						} `json:"type"`
					} `json:"fields"`
					EnumValues []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					} `json:"enumValues"`
				} `json:"types"`
			} `graphql:"__schema"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{}); err != nil {
			return nil, errors.Wrap(err, "failed to introspect GraphQL schema")
		}

		if format == "summary" {
			return formatSchemaSummary(&query.Schema)
		}

		return formatDetailedSchema(&query.Schema)
	})
}

// registerGetTypeDetailsTool registers the type details tool
func registerGetTypeDetailsTool(s *server.MCPServer) {
	typeDetailsTool := mcp.NewTool("get_graphql_type_details",
		mcp.WithDescription(`Get detailed information about a specific GraphQL type including its fields, arguments, and relationships.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GraphQL Type Details",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("type_name", mcp.Description("The name of the GraphQL type to get details for"), mcp.Required()),
	)

	s.AddTool(typeDetailsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		typeName, err := request.RequireString("type_name")
		if err != nil {
			return nil, err
		}

		var query struct {
			Type *struct {
				Name        string `json:"name"`
				Kind        string `json:"kind"`
				Description string `json:"description"`
				Fields      []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Type        struct {
						Name   string `json:"name"`
						Kind   string `json:"kind"`
						OfType *struct {
							Name string `json:"name"`
							Kind string `json:"kind"`
						} `json:"ofType"`
					} `json:"type"`
					Args []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name   string `json:"name"`
							Kind   string `json:"kind"`
							OfType *struct {
								Name string `json:"name"`
								Kind string `json:"kind"`
							} `json:"ofType"`
						} `json:"type"`
					} `json:"args"`
				} `json:"fields"`
				EnumValues []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"enumValues"`
				Interfaces []struct {
					Name string `json:"name"`
				} `json:"interfaces"`
				PossibleTypes []struct {
					Name string `json:"name"`
				} `json:"possibleTypes"`
			} `graphql:"__type(name: $name)"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{"name": graphql.String(typeName)}); err != nil {
			return nil, errors.Wrap(err, "failed to get type details")
		}

		if query.Type == nil {
			return mcp.NewToolResultText(fmt.Sprintf("Type '%s' not found in the GraphQL schema", typeName)), nil
		}

		typeJSON, err := json.MarshalIndent(query.Type, "", "  ")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal type details to JSON")
		}

		return mcp.NewToolResultText(string(typeJSON)), nil
	})
}

// registerSearchSchemaFieldsTool registers the schema field search tool
func registerSearchSchemaFieldsTool(s *server.MCPServer) {
	searchTool := mcp.NewTool("search_graphql_schema_fields",
		mcp.WithDescription(`Search for fields, types, or operations in the GraphQL schema. Useful for discovering available operations and finding specific functionality.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GraphQL Schema Fields",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("search_term", mcp.Description("The term to search for in field names, type names, and descriptions"), mcp.Required()),
		mcp.WithString("search_scope", mcp.Description("Scope of search: 'all', 'queries', 'mutations', 'types'"),
			mcp.DefaultString("all"),
			mcp.Enum("all", "queries", "mutations", "types")),
	)

	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		searchTerm, err := request.RequireString("search_term")
		if err != nil {
			return nil, err
		}

		searchScope := request.GetString("search_scope", "all")
		searchTerm = strings.ToLower(searchTerm)

		var query struct {
			Schema struct {
				QueryType struct {
					Name   string `json:"name"`
					Fields []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name string `json:"name"`
							Kind string `json:"kind"`
						} `json:"type"`
					} `json:"fields"`
				} `json:"queryType"`
				MutationType *struct {
					Name   string `json:"name"`
					Fields []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name string `json:"name"`
							Kind string `json:"kind"`
						} `json:"type"`
					} `json:"fields"`
				} `json:"mutationType"`
				Types []struct {
					Name        string `json:"name"`
					Kind        string `json:"kind"`
					Description string `json:"description"`
					Fields      []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
						Type        struct {
							Name string `json:"name"`
							Kind string `json:"kind"`
						} `json:"type"`
					} `json:"fields"`
				} `json:"types"`
			} `graphql:"__schema"`
		}

		if err := authenticated.Client().Query(ctx, &query, map[string]any{}); err != nil {
			return nil, errors.Wrap(err, "failed to introspect GraphQL schema")
		}

		results := searchSchemaFields(&query.Schema, searchTerm, searchScope)

		if len(results) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No results found for search term '%s' in scope '%s'", searchTerm, searchScope)), nil
		}

		resultsJSON, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal search results to JSON")
		}

		output := fmt.Sprintf("Found %d results for search term '%s' in scope '%s':\n\n%s",
			len(results), searchTerm, searchScope, string(resultsJSON))

		return mcp.NewToolResultText(output), nil
	})
}

// formatSchemaSummary formats the schema as a high-level summary
func formatSchemaSummary(schema any) (*mcp.CallToolResult, error) {
	schemaValue := reflect.ValueOf(schema).Elem()

	var summary strings.Builder
	summary.WriteString("GraphQL Schema Summary\n")
	summary.WriteString("======================\n\n")

	// Query Type
	queryType := schemaValue.FieldByName("QueryType")
	if queryType.IsValid() {
		fields := queryType.FieldByName("Fields")
		if fields.IsValid() && fields.Len() > 0 {
			summary.WriteString(fmt.Sprintf("Available Queries (%d):\n", fields.Len()))
			for i := 0; i < fields.Len(); i++ {
				field := fields.Index(i)
				name := field.FieldByName("Name").String()
				description := field.FieldByName("Description").String()
				if description != "" {
					summary.WriteString(fmt.Sprintf("  - %s: %s\n", name, description))
				} else {
					summary.WriteString(fmt.Sprintf("  - %s\n", name))
				}
			}
			summary.WriteString("\n")
		}
	}

	// Mutation Type
	mutationType := schemaValue.FieldByName("MutationType")
	if mutationType.IsValid() && !mutationType.IsNil() {
		fields := mutationType.Elem().FieldByName("Fields")
		if fields.IsValid() && fields.Len() > 0 {
			summary.WriteString(fmt.Sprintf("Available Mutations (%d):\n", fields.Len()))
			for i := 0; i < fields.Len(); i++ {
				field := fields.Index(i)
				name := field.FieldByName("Name").String()
				description := field.FieldByName("Description").String()
				if description != "" {
					summary.WriteString(fmt.Sprintf("  - %s: %s\n", name, description))
				} else {
					summary.WriteString(fmt.Sprintf("  - %s\n", name))
				}
			}
			summary.WriteString("\n")
		}
	}

	// Types
	types := schemaValue.FieldByName("Types")
	if types.IsValid() && types.Len() > 0 {
		var objectTypes, enumTypes, scalarTypes []string

		for i := 0; i < types.Len(); i++ {
			typeItem := types.Index(i)
			name := typeItem.FieldByName("Name").String()
			kind := typeItem.FieldByName("Kind").String()

			// Skip built-in types
			if strings.HasPrefix(name, "__") {
				continue
			}

			switch kind {
			case "OBJECT":
				objectTypes = append(objectTypes, name)
			case "ENUM":
				enumTypes = append(enumTypes, name)
			case "SCALAR":
				scalarTypes = append(scalarTypes, name)
			}
		}

		if len(objectTypes) > 0 {
			sort.Strings(objectTypes)
			summary.WriteString(fmt.Sprintf("Object Types (%d): %s\n\n", len(objectTypes), strings.Join(objectTypes, ", ")))
		}

		if len(enumTypes) > 0 {
			sort.Strings(enumTypes)
			summary.WriteString(fmt.Sprintf("Enum Types (%d): %s\n\n", len(enumTypes), strings.Join(enumTypes, ", ")))
		}

		if len(scalarTypes) > 0 {
			sort.Strings(scalarTypes)
			summary.WriteString(fmt.Sprintf("Scalar Types (%d): %s\n\n", len(scalarTypes), strings.Join(scalarTypes, ", ")))
		}
	}

	return mcp.NewToolResultText(summary.String()), nil
}

// formatDetailedSchema formats the complete schema with all details
func formatDetailedSchema(schema any) (*mcp.CallToolResult, error) {
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal schema to JSON")
	}

	return mcp.NewToolResultText(string(schemaJSON)), nil
}

// searchSchemaFields searches for fields matching the search term
func searchSchemaFields(schema any, searchTerm, searchScope string) []map[string]any {
	var results []map[string]any
	schemaValue := reflect.ValueOf(schema).Elem()

	// Search in queries
	if searchScope == "all" || searchScope == "queries" {
		queryType := schemaValue.FieldByName("QueryType")
		if queryType.IsValid() {
			results = append(results, searchInFields(queryType.FieldByName("Fields"), searchTerm, "Query")...)
		}
	}

	// Search in mutations
	if searchScope == "all" || searchScope == "mutations" {
		mutationType := schemaValue.FieldByName("MutationType")
		if mutationType.IsValid() && !mutationType.IsNil() {
			results = append(results, searchInFields(mutationType.Elem().FieldByName("Fields"), searchTerm, "Mutation")...)
		}
	}

	// Search in types
	if searchScope == "all" || searchScope == "types" {
		types := schemaValue.FieldByName("Types")
		if types.IsValid() && types.Len() > 0 {
			for i := 0; i < types.Len(); i++ {
				typeItem := types.Index(i)
				name := typeItem.FieldByName("Name").String()
				kind := typeItem.FieldByName("Kind").String()
				description := typeItem.FieldByName("Description").String()

				// Skip built-in types
				if strings.HasPrefix(name, "__") {
					continue
				}

				// Check if type name or description matches
				if strings.Contains(strings.ToLower(name), searchTerm) ||
					strings.Contains(strings.ToLower(description), searchTerm) {
					results = append(results, map[string]any{
						"category":    "Type",
						"type":        kind,
						"name":        name,
						"description": description,
					})
				}

				// Search in type fields
				fields := typeItem.FieldByName("Fields")
				if fields.IsValid() && fields.Len() > 0 {
					typeResults := searchInFields(fields, searchTerm, fmt.Sprintf("Type.%s", name))
					results = append(results, typeResults...)
				}
			}
		}
	}

	return results
}

// searchInFields searches for fields matching the search term in a fields slice
func searchInFields(fields reflect.Value, searchTerm, category string) []map[string]any {
	var results []map[string]any

	if !fields.IsValid() || fields.Len() == 0 {
		return results
	}

	for i := 0; i < fields.Len(); i++ {
		field := fields.Index(i)
		name := field.FieldByName("Name").String()
		description := field.FieldByName("Description").String()

		// Check if field name or description matches
		if strings.Contains(strings.ToLower(name), searchTerm) ||
			strings.Contains(strings.ToLower(description), searchTerm) {

			// Get type information
			typeField := field.FieldByName("Type")
			var typeName string
			if typeField.IsValid() {
				typeName = typeField.FieldByName("Name").String()
			}

			results = append(results, map[string]any{
				"category":    category,
				"name":        name,
				"description": description,
				"type":        typeName,
			})
		}
	}

	return results
}

// registerAuthenticationGuideTool registers the authentication guide tool
func registerAuthenticationGuideTool(s *server.MCPServer) {
	authTool := mcp.NewTool("get_authentication_guide",
		mcp.WithDescription(`Get comprehensive guidance on how to authenticate with the Spacelift GraphQL API, including all available authentication methods and practical examples.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Authentication Guide",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("auth_method", mcp.Description("Specific authentication method to focus on: 'all', 'api_key', 'api_token', 'github_token', 'cli_token'"),
			mcp.DefaultString("all"),
			mcp.Enum("all", "api_key", "api_token", "github_token", "cli_token")),
	)

	s.AddTool(authTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		authMethod := request.GetString("auth_method", "all")

		var guide strings.Builder
		guide.WriteString("Spacelift GraphQL API Authentication Guide\n")
		guide.WriteString("==========================================\n\n")

		guide.WriteString("GraphQL Endpoint: https://<account-name>.app.spacelift.io/graphql\n")
		guide.WriteString("Authentication: JWT Bearer Token (expires after 1 hour)\n")
		guide.WriteString("Authorization Header: Bearer <jwt-token>\n")
		guide.WriteString("Content-Type: application/json\n\n")

		if authMethod == "all" || authMethod == "api_key" {
			guide.WriteString("1. API KEY AUTHENTICATION (Recommended)\n")
			guide.WriteString("========================================\n\n")
			guide.WriteString("Overview:\n")
			guide.WriteString("- Exchange API Key ID and Secret for JWT token via GraphQL mutation\n")
			guide.WriteString("- Create API keys in Organization Settings > API Keys\n")
			guide.WriteString("- Supports space-level and organization-level access control\n")
			guide.WriteString("- JWT tokens expire after 1 hour and need to be refreshed\n\n")

			guide.WriteString("Step 1: Exchange API Key for JWT Token\n")
			guide.WriteString("---------------------------------------\n")
			guide.WriteString("POST https://<account>.app.spacelift.io/graphql\n")
			guide.WriteString("Content-Type: application/json\n\n")
			guide.WriteString("Request Body:\n")
			guide.WriteString("{\n")
			guide.WriteString("  \"query\": \"mutation($id: ID!, $secret: String!) { apiKeyUser(id: $id, secret: $secret) { jwt validUntil } }\",\n")
			guide.WriteString("  \"variables\": {\n")
			guide.WriteString("    \"id\": \"<your-api-key-id>\",\n")
			guide.WriteString("    \"secret\": \"<your-api-key-secret>\"\n")
			guide.WriteString("  }\n")
			guide.WriteString("}\n\n")
			guide.WriteString("Response:\n")
			guide.WriteString("{\n")
			guide.WriteString("  \"data\": {\n")
			guide.WriteString("    \"apiKeyUser\": {\n")
			guide.WriteString("      \"jwt\": \"<jwt-token>\",\n")
			guide.WriteString("      \"validUntil\": 1234567890\n")
			guide.WriteString("    }\n")
			guide.WriteString("  }\n")
			guide.WriteString("}\n\n")

			guide.WriteString("Step 2: Use JWT Token for API Calls\n")
			guide.WriteString("------------------------------------\n")
			guide.WriteString("POST https://<account>.app.spacelift.io/graphql\n")
			guide.WriteString("Authorization: Bearer <jwt-token>\n")
			guide.WriteString("Content-Type: application/json\n\n")
			guide.WriteString("Request Body:\n")
			guide.WriteString("{\n")
			guide.WriteString("  \"query\": \"{ stacks { id name } }\"\n")
			guide.WriteString("}\n\n")

			guide.WriteString("Important Notes:\n")
			guide.WriteString("- Store JWT token securely and refresh before expiry (validUntil timestamp)\n")
			guide.WriteString("- Recommend refreshing 30 seconds before expiry to avoid race conditions\n")
			guide.WriteString("- Handle 401 Unauthorized responses by re-exchanging API key for new JWT\n\n")
		}

		if authMethod == "all" || authMethod == "github_token" {
			guide.WriteString("2. GITHUB TOKEN AUTHENTICATION (GitHub SSO Only)\n")
			guide.WriteString("=================================================\n\n")
			guide.WriteString("Overview:\n")
			guide.WriteString("- Exchange GitHub Personal Access Token for JWT token\n")
			guide.WriteString("- Only available when GitHub SSO is configured for your account\n")
			guide.WriteString("- Requires 'read:user' scope on GitHub token\n\n")

			guide.WriteString("Step 1: Exchange GitHub Token for JWT Token\n")
			guide.WriteString("--------------------------------------------\n")
			guide.WriteString("POST https://<account>.app.spacelift.io/graphql\n")
			guide.WriteString("Content-Type: application/json\n\n")
			guide.WriteString("Request Body:\n")
			guide.WriteString("{\n")
			guide.WriteString("  \"query\": \"mutation($token: String!) { oauthUser(token: $token) { jwt validUntil } }\",\n")
			guide.WriteString("  \"variables\": {\n")
			guide.WriteString("    \"token\": \"<github-personal-access-token>\"\n")
			guide.WriteString("  }\n")
			guide.WriteString("}\n\n")
			guide.WriteString("Response: Same as API Key authentication\n\n")
		}

		if authMethod == "all" || authMethod == "api_token" {
			guide.WriteString("3. DIRECT JWT TOKEN AUTHENTICATION\n")
			guide.WriteString("===================================\n\n")
			guide.WriteString("Overview:\n")
			guide.WriteString("- Use a pre-obtained JWT token directly\n")
			guide.WriteString("- No exchange needed, but tokens still expire after 1 hour\n")
			guide.WriteString("- Useful when JWT token is obtained through other means\n\n")

			guide.WriteString("Usage:\n")
			guide.WriteString("------\n")
			guide.WriteString("POST https://<account>.app.spacelift.io/graphql\n")
			guide.WriteString("Authorization: Bearer <jwt-token>\n")
			guide.WriteString("Content-Type: application/json\n\n")
			guide.WriteString("Request Body:\n")
			guide.WriteString("{\n")
			guide.WriteString("  \"query\": \"{ stacks { id name } }\"\n")
			guide.WriteString("}\n\n")
		}

		if authMethod == "all" || authMethod == "cli_token" {
			guide.WriteString("4. SPACECTL CLI TOKEN\n")
			guide.WriteString("=====================\n\n")
			guide.WriteString("Overview:\n")
			guide.WriteString("- Use spacectl CLI to authenticate and export tokens\n")
			guide.WriteString("- Handles token refresh automatically\n")
			guide.WriteString("- Stores credentials in ~/.spacelift/ directory\n\n")

			guide.WriteString("Commands:\n")
			guide.WriteString("---------\n")
			guide.WriteString("# Authenticate with spacectl\n")
			guide.WriteString("spacectl profile login\n\n")
			guide.WriteString("# Export current token for external use\n")
			guide.WriteString("spacectl profile export-token\n\n")
		}

		if authMethod == "all" {
			guide.WriteString("LANGUAGE-SPECIFIC EXAMPLES\n")
			guide.WriteString("==========================\n\n")

			guide.WriteString("Python Example:\n")
			guide.WriteString("---------------\n")
			guide.WriteString("import requests\nimport json\nimport time\n\n")
			guide.WriteString("class SpaceliftClient:\n")
			guide.WriteString("    def __init__(self, endpoint, api_key_id, api_key_secret):\n")
			guide.WriteString("        self.endpoint = endpoint\n")
			guide.WriteString("        self.api_key_id = api_key_id\n")
			guide.WriteString("        self.api_key_secret = api_key_secret\n")
			guide.WriteString("        self.jwt_token = None\n")
			guide.WriteString("        self.token_expires_at = 0\n\n")
			guide.WriteString("    def authenticate(self):\n")
			guide.WriteString("        if self.jwt_token and time.time() < self.token_expires_at - 30:\n")
			guide.WriteString("            return self.jwt_token\n")
			guide.WriteString("        \n")
			guide.WriteString("        mutation = '''\n")
			guide.WriteString("        mutation($id: ID!, $secret: String!) {\n")
			guide.WriteString("            apiKeyUser(id: $id, secret: $secret) {\n")
			guide.WriteString("                jwt\n")
			guide.WriteString("                validUntil\n")
			guide.WriteString("            }\n")
			guide.WriteString("        }\n")
			guide.WriteString("        '''\n")
			guide.WriteString("        \n")
			guide.WriteString("        response = requests.post(\n")
			guide.WriteString("            f'{self.endpoint}/graphql',\n")
			guide.WriteString("            json={\n")
			guide.WriteString("                'query': mutation,\n")
			guide.WriteString("                'variables': {\n")
			guide.WriteString("                    'id': self.api_key_id,\n")
			guide.WriteString("                    'secret': self.api_key_secret\n")
			guide.WriteString("                }\n")
			guide.WriteString("            }\n")
			guide.WriteString("        )\n")
			guide.WriteString("        \n")
			guide.WriteString("        data = response.json()\n")
			guide.WriteString("        self.jwt_token = data['data']['apiKeyUser']['jwt']\n")
			guide.WriteString("        self.token_expires_at = data['data']['apiKeyUser']['validUntil']\n")
			guide.WriteString("        return self.jwt_token\n\n")
			guide.WriteString("    def query(self, query_str, variables=None):\n")
			guide.WriteString("        token = self.authenticate()\n")
			guide.WriteString("        return requests.post(\n")
			guide.WriteString("            f'{self.endpoint}/graphql',\n")
			guide.WriteString("            headers={'Authorization': f'Bearer {token}'},\n")
			guide.WriteString("            json={'query': query_str, 'variables': variables or {}}\n")
			guide.WriteString("        ).json()\n\n")

			guide.WriteString("Node.js Example:\n")
			guide.WriteString("----------------\n")
			guide.WriteString("const axios = require('axios');\n\n")
			guide.WriteString("class SpaceliftClient {\n")
			guide.WriteString("  constructor(endpoint, apiKeyId, apiKeySecret) {\n")
			guide.WriteString("    this.endpoint = endpoint;\n")
			guide.WriteString("    this.apiKeyId = apiKeyId;\n")
			guide.WriteString("    this.apiKeySecret = apiKeySecret;\n")
			guide.WriteString("    this.jwtToken = null;\n")
			guide.WriteString("    this.tokenExpiresAt = 0;\n")
			guide.WriteString("  }\n\n")
			guide.WriteString("  async authenticate() {\n")
			guide.WriteString("    if (this.jwtToken && Date.now() / 1000 < this.tokenExpiresAt - 30) {\n")
			guide.WriteString("      return this.jwtToken;\n")
			guide.WriteString("    }\n")
			guide.WriteString("    \n")
			guide.WriteString("    const mutation = `\n")
			guide.WriteString("      mutation($id: ID!, $secret: String!) {\n")
			guide.WriteString("        apiKeyUser(id: $id, secret: $secret) {\n")
			guide.WriteString("          jwt\n")
			guide.WriteString("          validUntil\n")
			guide.WriteString("        }\n")
			guide.WriteString("      }\n")
			guide.WriteString("    `;\n")
			guide.WriteString("    \n")
			guide.WriteString("    const response = await axios.post(`${this.endpoint}/graphql`, {\n")
			guide.WriteString("      query: mutation,\n")
			guide.WriteString("      variables: {\n")
			guide.WriteString("        id: this.apiKeyId,\n")
			guide.WriteString("        secret: this.apiKeySecret\n")
			guide.WriteString("      }\n")
			guide.WriteString("    });\n")
			guide.WriteString("    \n")
			guide.WriteString("    const { jwt, validUntil } = response.data.data.apiKeyUser;\n")
			guide.WriteString("    this.jwtToken = jwt;\n")
			guide.WriteString("    this.tokenExpiresAt = validUntil;\n")
			guide.WriteString("    return jwt;\n")
			guide.WriteString("  }\n\n")
			guide.WriteString("  async query(queryStr, variables = {}) {\n")
			guide.WriteString("    const token = await this.authenticate();\n")
			guide.WriteString("    const response = await axios.post(`${this.endpoint}/graphql`, {\n")
			guide.WriteString("      query: queryStr,\n")
			guide.WriteString("      variables\n")
			guide.WriteString("    }, {\n")
			guide.WriteString("      headers: {\n")
			guide.WriteString("        'Authorization': `Bearer ${token}`\n")
			guide.WriteString("      }\n")
			guide.WriteString("    });\n")
			guide.WriteString("    return response.data;\n")
			guide.WriteString("  }\n")
			guide.WriteString("}\n\n")

			guide.WriteString("BEST PRACTICES\n")
			guide.WriteString("==============\n\n")
			guide.WriteString("1. Token Management:\n")
			guide.WriteString("   - Cache JWT tokens to avoid unnecessary API calls\n")
			guide.WriteString("   - Refresh tokens 30 seconds before expiry\n")
			guide.WriteString("   - Handle 401 Unauthorized by re-authenticating\n\n")
			guide.WriteString("2. Error Handling:\n")
			guide.WriteString("   - Check for 'errors' field in GraphQL responses\n")
			guide.WriteString("   - Implement retry logic for network failures\n")
			guide.WriteString("   - Log authentication failures for debugging\n\n")
			guide.WriteString("3. Security:\n")
			guide.WriteString("   - Store API keys and secrets securely (environment variables, key vaults)\n")
			guide.WriteString("   - Never commit credentials to version control\n")
			guide.WriteString("   - Use HTTPS for all API communications\n")
			guide.WriteString("   - Implement proper credential rotation policies\n\n")
			guide.WriteString("4. Rate Limiting:\n")
			guide.WriteString("   - Implement exponential backoff for failed requests\n")
			guide.WriteString("   - Respect any rate limiting headers returned by the API\n")
			guide.WriteString("   - Use batch queries when possible to reduce API calls\n")
		}

		return mcp.NewToolResultText(guide.String()), nil
	})
}
