package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	versioncmd "github.com/spacelift-io/spacectl/internal/cmd/version"
)

var version = "dev"
var date = "2006-01-02T15:04:05Z"

func main() {
	compileTime, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not parse compilation date: %v", err))
	}
	app := &cli.App{
		Name:     "spacectl",
		Version:  version,
		Compiled: compileTime,
		Usage:    "Programmatic access to Spacelift GraphQL API.",
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
