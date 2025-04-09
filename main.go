package main

import (
	"log"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/completion"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	versioncmd "github.com/spacelift-io/spacectl/internal/cmd/version"
	"github.com/spacelift-io/spacectl/internal/cmd/whoami"
	"github.com/urfave/cli/v2"
)

var version = "dev"
var date = "2006-01-02T15:04:05Z"

func main() {
	compileTime, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Fatalf("Could not parse compilation date: %v", err)
	}

	// TODO: query for the actual version
	instanceVersion := cmd.SpaceliftInstanceVersion{
		SaaS:    false,
		Version: semver.MustParse("2.4.0"),
	}

	app := &cli.App{
		Name:                 "spacectl",
		Version:              version,
		Compiled:             compileTime,
		Usage:                "Programmatic access to Spacelift GraphQL API.",
		EnableBashCompletion: true,
		Commands: append([]*cli.Command{
			profile.Command(),
			whoami.Command(),
			versioncmd.Command(version),
			completion.Command(),
		}, cmd.ResolveCommands(instanceVersion, []cmd.Command{
			module.Command(),
			stack.Command(),
			// provider.Command(),
			// runexternaldependency.Command(),
			// workerpools.Command(),
			// blueprint.Command(),
			// policy.Command(),
			// audittrail.Command(),
		})...),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
