package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/completion"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/provider"
	runexternaldependency "github.com/spacelift-io/spacectl/internal/cmd/run_external_dependency"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	versioncmd "github.com/spacelift-io/spacectl/internal/cmd/version"
	"github.com/spacelift-io/spacectl/internal/cmd/whoami"
	"github.com/spacelift-io/spacectl/internal/cmd/workerpools"
)

var version = "dev"
var date = "2006-01-02T15:04:05Z"

func main() {
	compileTime, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Fatalf("Could not parse compilation date: %v", err)
	}
	app := &cli.App{
		Name:                 "spacectl",
		Version:              version,
		Compiled:             compileTime,
		Usage:                "Programmatic access to Spacelift GraphQL API.",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			module.Command(),
			profile.Command(),
			provider.Command(),
			runexternaldependency.Command(),
			stack.Command(),
			whoami.Command(),
			versioncmd.Command(version),
			workerpools.Command(),
			completion.Command(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
