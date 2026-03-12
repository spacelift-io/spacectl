package api

import (
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var (
	flagQuery = &cli.StringFlag{
		Name:  "query",
		Usage: "GraphQL query or mutation string. Use '-' to read from stdin",
	}
	flagFile = &cli.StringFlag{
		Name:  "file",
		Usage: "Path to a file containing a GraphQL query or mutation",
	}
	flagVariables = &cli.StringFlag{
		Name:  "variables",
		Usage: "JSON object with GraphQL variables",
	}
	flagOperation = &cli.StringFlag{
		Name:  "operation",
		Usage: "Optional GraphQL operation name",
	}
	flagRaw = &cli.BoolFlag{
		Name:  "raw",
		Usage: "Output response body without pretty-printing",
	}
	flagSchema = &cli.BoolFlag{
		Name:  "schema",
		Usage: "Download the GraphQL schema via introspection or list schema items",
	}
)

// Command returns the api command definition.
func Command() cmd.Command {
	return cmd.Command{
		Name:     "api",
		Usage:    "Call the Spacelift GraphQL API",
		Category: "GraphQL",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command: &cli.Command{
					Flags: []cli.Flag{
						flagQuery,
						flagFile,
						flagVariables,
						flagOperation,
						flagRaw,
						flagSchema,
					},
					Description: "You can also pass a query as a positional argument. If it does not start with `query`, `mutation`, `subscription`, or `{`, it will be wrapped as `query { ... }`. When `--schema` is set, the positional argument selects what to show: `queries`, `mutations`, `types`, or a specific query/mutation/type name.",
					Before:      authenticated.Ensure,
					Action:      run,
					ArgsUsage:   "[query]",
				},
			},
		},
	}
}
