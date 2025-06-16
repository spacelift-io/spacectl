# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Build and Development
- `go build` - Build the spacectl binary
- `make lint` - Run golangci-lint for code linting
- `go test ./...` - Run all tests
- `go run main.go` - Run spacectl from source

### Testing
- `go test ./internal/cmd/...` - Run tests for command packages
- `go test -v ./client/session/...` - Run session tests with verbose output

## Architecture Overview

### Core Components

**CLI Framework**: Built on `github.com/urfave/cli/v3` with a modular command structure

**GraphQL Client**: Uses `github.com/shurcooL/graphql` (forked at `github.com/spacelift-io/graphql`) for API communication

**Authentication System**: Multi-method authentication supporting:
- Environment variables (API tokens, GitHub tokens, API keys)  
- Profile-based credentials stored in `~/.spacelift/`
- Session management in `client/session/`

**Command Versioning**: Commands support multiple versions for compatibility between SaaS and Self-Hosted Spacelift instances using `cmd.VersionedCommand`

### Key Packages

- `main.go` - Entry point, command registration, instance version detection
- `internal/cmd/` - All CLI commands organized by functionality
- `client/` - GraphQL client and HTTP client management
- `client/session/` - Authentication and session handling
- `internal/cmd/authenticated/` - Shared authentication logic for commands

### Command Structure

Commands are defined using `cmd.Command` with versioned implementations:
- `EarliestVersion: cmd.SupportedVersionAll` - Works with any Spacelift version
- `EarliestVersion: cmd.SupportedVersionLatest` - SaaS only, pending Self-Hosted release
- `EarliestVersion: cmd.SupportedVersion("2.5.0")` - Requires specific version or higher

### Error Handling

The client automatically detects unauthorized errors and provides contextual messages:
- Distinguishes between "no access to resource" vs "need to re-login"
- Uses `spacectl profile login` for authentication guidance

### MCP Server Integration

Includes Model Context Protocol server (`mcp.Command()`) for AI model interaction with Spacelift resources.

## Go Development Tools

When working with this Go codebase, make liberal use of available MCP servers:

- **mcp-gopls**: Essential for spacectl Go development, providing:
  - **Navigate CLI Architecture**: Use `GoToDefinition` and `FindReferences` to trace command implementations from `internal/cmd/` through to GraphQL client calls
  - **Debug Authentication Flow**: Search symbols like "session", "profile", "token" to understand the multi-method auth system in `client/session/`
  - **Explore GraphQL Integration**: Find implementers of GraphQL query structs and trace data flow from API responses to CLI output
  - **Maintain Code Quality**: `GetDiagnostics` catches Go compilation errors, `FormatCode` ensures gofmt compliance, `OrganizeImports` manages the extensive import dependencies
  - **Refactor Safely**: `RenameSymbol` for renaming across the modular command structure without breaking CLI interface compatibility
  - **Understand Command Versioning**: Search for `VersionedCommand` usage to see how commands support different Spacelift instance versions
- **context7**: Essential for accessing current documentation for spacectl's key dependencies:
  - **CLI Framework**: Get latest `github.com/urfave/cli/v3` patterns for command structure, flags, and subcommands used throughout `internal/cmd/`
  - **GraphQL Client**: Access `github.com/shurcooL/graphql` (forked as `spacelift-io/graphql`) documentation for query building and response handling
  - **Terminal UI**: Find examples for `github.com/charmbracelet/bubbletea`, `bubbles`, and `lipgloss` used in interactive features like local preview and worker pool management
  - **Authentication Libraries**: Get guidance on `golang.org/x/oauth2` and `github.com/golang-jwt/jwt/v4` for the multi-method auth system
  - **Utility Libraries**: Access docs for `github.com/manifoldco/promptui` (user prompts), `github.com/pkg/browser` (opening URLs), `github.com/mholt/archiver/v3` (file handling)
  - **Testing Frameworks**: Get current patterns for `github.com/stretchr/testify` and `github.com/onsi/gomega` used in the test suite
- **spacectl**: For working with Spacelift's GraphQL schema and API operations. Use this MCP server to:
  - Introspect the GraphQL schema (`mcp__spacectl__introspect_graphql_schema`)
  - Search for specific GraphQL fields and types (`mcp__spacectl__search_graphql_schema_fields`)
  - Get detailed information about GraphQL types (`mcp__spacectl__get_graphql_type_details`)
  - Get comprehensive authentication guidance (`mcp__spacectl__get_authentication_guide`)

These tools provide enhanced Go development capabilities including real-time code analysis, intelligent code completion, comprehensive documentation access, and direct Spacelift API interaction.
