package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/apikey"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/blueprint"
	spaceliftcontext "github.com/spacelift-io/spacectl/internal/cmd/context"
	"github.com/spacelift-io/spacectl/internal/cmd/graphql"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/policy"
	"github.com/spacelift-io/spacectl/internal/cmd/space"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	"github.com/spacelift-io/spacectl/internal/cmd/workerpool"
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
								policy.RegisterMCPTools(s)
								graphql.RegisterMCPTools(s)
								spaceliftcontext.RegisterMCPTools(s)
								apikey.RegisterMCPTools(s)
								space.RegisterMCPTools(s)
								workerpool.RegisterMCPTools(s)
								blueprint.RegisterMCPTools(s)

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
								policy.RegisterMCPTools(s)
								graphql.RegisterMCPTools(s)
								spaceliftcontext.RegisterMCPTools(s)
								apikey.RegisterMCPTools(s)
								space.RegisterMCPTools(s)
								workerpool.RegisterMCPTools(s)
								blueprint.RegisterMCPTools(s)

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
		server.WithLogging(),
		server.WithRecovery(),
	)

	return s
}
