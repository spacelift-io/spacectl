package graphql

import (
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterMCPTools(t *testing.T) {
	s := server.NewMCPServer(
		"Test MCP Server",
		"1.0.0",
	)

	// This should not panic
	RegisterMCPTools(s)

	// Verify that tools were registered by checking tool count
	// Since the server doesn't expose registered tools directly,
	// we just ensure registration doesn't panic
	if s == nil {
		t.Fatal("MCP server should not be nil after registration")
	}
}

func TestSearchSchemaFields(t *testing.T) {
	// Create a mock schema structure for testing
	mockSchema := struct {
		QueryType struct {
			Fields []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"fields"`
		} `json:"queryType"`
		Types []struct {
			Name        string `json:"name"`
			Kind        string `json:"kind"`
			Description string `json:"description"`
			Fields      []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"fields"`
		} `json:"types"`
	}{}

	// Add some test data
	mockSchema.QueryType.Fields = []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}{
		{Name: "stacks", Description: "List all stacks"},
		{Name: "modules", Description: "List all modules"},
	}

	mockSchema.Types = []struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
		Fields      []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"fields"`
	}{
		{
			Name:        "Stack",
			Kind:        "OBJECT",
			Description: "A Spacelift stack",
			Fields: []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}{
				{Name: "id", Description: "Stack ID"},
				{Name: "name", Description: "Stack name"},
			},
		},
	}

	// Test search functionality
	results := searchSchemaFields(&mockSchema, "stack", "all")

	if len(results) == 0 {
		t.Error("Expected to find results for 'stack' search term")
	}

	// Check that we found the expected results
	foundQuery := false
	foundType := false

	for _, result := range results {
		if result["category"] == "Query" && result["name"] == "stacks" {
			foundQuery = true
		}
		if result["category"] == "Type" && result["name"] == "Stack" {
			foundType = true
		}
	}

	if !foundQuery {
		t.Error("Expected to find 'stacks' query in results")
	}
	if !foundType {
		t.Error("Expected to find 'Stack' type in results")
	}
}
