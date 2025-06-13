package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
)

// Command returns the MCP command subtree.
func Command() cmd.Command {
	return cmd.Command{
		Name:  "mcp",
		Usage: "Manage MCP server",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command:         &cli.Command{},
			},
		},
		Subcommands: []cmd.Command{
			{
				Name:  "server",
				Usage: "Start MCP server",
				Versions: []cmd.VersionedCommand{
					{
						EarliestVersion: cmd.SupportedVersionAll,
						Command: &cli.Command{
							ArgsUsage: cmd.EmptyArgsUsage,
							Action: func(_ context.Context, _ *cli.Command) error {
								s := mcpServer()

								stack.RegisterMCPTools(s, stack.McpOptions{
									UseHeadersForLocalPreview: false,
								})
								module.RegisterMCPTools(s)

								return server.ServeStdio(s)
							},
							Before: authenticated.Ensure,
						},
					},
					{
						EarliestVersion: cmd.SupportedVersion("2.5.0"),
						Command: &cli.Command{
							ArgsUsage: cmd.EmptyArgsUsage,
							Action: func(_ context.Context, _ *cli.Command) error {
								s := mcpServer()

								stack.RegisterMCPTools(s, stack.McpOptions{
									UseHeadersForLocalPreview: true,
								})
								module.RegisterMCPTools(s)

								return server.ServeStdio(s)
							},
							Before: authenticated.Ensure,
						},
					},
				},
			},
		},
	}
}

func mcpServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Spacelift MCP Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	return s
}
