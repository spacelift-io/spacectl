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

- **mcp-gopls**: For Go language server features including code navigation, symbol search, diagnostics, formatting, and refactoring
- **context7**: For accessing up-to-date documentation and examples for Go packages and frameworks used in this project
- **mcp-graphql**: For interacting with Spacelift's GraphQL schema at <https://demo.app.spacelift.io/graphql> - use this to introspect the schema and execute GraphQL queries for development assistance

These tools provide enhanced Go development capabilities including real-time code analysis, intelligent code completion, comprehensive documentation access, and direct GraphQL API interaction.
