package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:     "spacectl",
		Version:  version,
		Compiled: time.Now(),
		Usage:    "Programmatic access to Spacelift GraphQL API.",
		Commands: []*cli.Command{
			profile.Command(),
			stack.Command(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
