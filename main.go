package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/audittrail"
	"github.com/spacelift-io/spacectl/internal/cmd/blueprint"
	"github.com/spacelift-io/spacectl/internal/cmd/completion"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/policy"
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

type debugInfoQuery struct {
	DebugInfo struct {
		SelfHostedVersion string `graphql:"selfHostedVersion"`
	} `graphql:"debugInfo"`
}

func getSpaceliftInstanceVersion() cmd.SpaceliftInstanceVersion {
	instanceVersion := cmd.SpaceliftInstanceVersion{
		InstanceType: cmd.SpaceliftInstanceTypeUnknown,
	}

	ctx, httpClient := session.Defaults()

	// Create a new session - this may fail if the user doesn't have valid credentials.
	// In that case we just treat the version as unknown.
	sess, err := session.New(ctx, httpClient)
	if err != nil {
		return instanceVersion
	}

	c := client.New(httpClient, sess)

	// Query the GraphQL API for the actual version. If this fails, we also treat the
	// version as unknown.
	var query debugInfoQuery
	if err := c.Query(context.Background(), &query, nil); err != nil {
		return instanceVersion
	}

	// If the query succeeds, determine if this is a SaaS or Self-Hosted instance
	if query.DebugInfo.SelfHostedVersion == "" {
		// Empty version means SaaS
		instanceVersion.InstanceType = cmd.SpaceliftInstanceTypeSaaS
	} else {
		// Non-empty version means Self-Hosted
		instanceVersion.InstanceType = cmd.SpaceliftInstanceTypeSelfHosted

		v, err := semver.NewVersion(query.DebugInfo.SelfHostedVersion)
		if err == nil {
			instanceVersion.Version = v
		} else {
			log.Printf("Warning: Failed to parse Self-Hosted version string: %q: %v",
				query.DebugInfo.SelfHostedVersion, err)
		}
	}

	return instanceVersion
}

func main() {
	compileTime, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Fatalf("Could not parse compilation date: %v", err)
	}

	instanceVersion := getSpaceliftInstanceVersion()

	if instanceVersion.InstanceType == cmd.SpaceliftInstanceTypeUnknown {
		log.Println("Warning: Unable to determine Spacelift instance type. Some commands may be unavailable until you authenticate with Spacelift.")
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
			versioncmd.Command(version, instanceVersion),
			completion.Command(),
		}, cmd.ResolveCommands(instanceVersion, []cmd.Command{
			module.Command(),
			stack.Command(),
			provider.Command(),
			runexternaldependency.Command(),
			workerpools.Command(),
			blueprint.Command(),
			policy.Command(),
			audittrail.Command(),
		})...),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
