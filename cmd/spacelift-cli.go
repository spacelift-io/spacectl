package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/cmd/internal/profile"
	"github.com/spacelift-io/spacelift-cli/cmd/internal/stack"
)

func main() {
	app := &cli.App{
		Name:  "spacelift-cli",
		Usage: "Programmatic access to Spacelift GraphQL API",
		Commands: []*cli.Command{
			profile.Command(),
			stack.Command(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
