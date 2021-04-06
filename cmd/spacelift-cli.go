package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacelift-cli/cmd/internal/account"
)

func main() {
	app := &cli.App{
		Name:     "spacelift-cli",
		Usage:    "Programmatic access to Spacelift GraphQL API",
		Commands: []*cli.Command{account.Command()},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
