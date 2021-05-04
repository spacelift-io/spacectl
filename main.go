package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	versioncmd "github.com/spacelift-io/spacectl/internal/cmd/version"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:  "spacectl",
		Usage: "Programmatic access to Spacelift GraphQL API",
		Commands: []*cli.Command{
			profile.Command(),
			stack.Command(),
			versioncmd.Command(version),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
