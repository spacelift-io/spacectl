package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
)

func Command() cmd.Command {
	return cmd.Command{
		Name:  "mcp",
		Usage: "Start MCP server",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command: &cli.Command{
					Action: func(cliCtx *cli.Context) error {
						s := server.NewMCPServer(
							"Spacelift MCP Server",
							"1.0.0",
							server.WithResourceCapabilities(true, true),
							server.WithLogging(),
							server.WithRecovery(),
						)

						stack.RegisterMCPTools(s)

						fmt.Println("Starting MCP server...")
						return server.ServeStdio(s)
					},
					Before:    authenticated.Ensure,
					ArgsUsage: cmd.EmptyArgsUsage,
				},
			},
		},
	}
}
