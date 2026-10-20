package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/ai"
	"github.com/spacelift-io/spacectl/internal/cmd/api"
	"github.com/spacelift-io/spacectl/internal/cmd/audittrail"
	"github.com/spacelift-io/spacectl/internal/cmd/blueprint"
	"github.com/spacelift-io/spacectl/internal/cmd/mcp"
	"github.com/spacelift-io/spacectl/internal/cmd/module"
	"github.com/spacelift-io/spacectl/internal/cmd/policy"
	"github.com/spacelift-io/spacectl/internal/cmd/profile"
	"github.com/spacelift-io/spacectl/internal/cmd/provider"
	"github.com/spacelift-io/spacectl/internal/cmd/registry"
	runexternaldependency "github.com/spacelift-io/spacectl/internal/cmd/run_external_dependency"
	"github.com/spacelift-io/spacectl/internal/cmd/stack"
	"github.com/spacelift-io/spacectl/internal/cmd/template"
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

// getSpaceliftInstanceVersion detects the type and version of the Spacelift
// instance the current credentials point at. When detection fails the type is
// unknown and the returned error says why.
func getSpaceliftInstanceVersion(ctx context.Context) (cmd.SpaceliftInstanceVersion, error) {
	instanceVersion := cmd.SpaceliftInstanceVersion{
		InstanceType: cmd.SpaceliftInstanceTypeUnknown,
	}

	httpClient, err := client.GetHTTPClient()
	if err != nil {
		return instanceVersion, err
	}

	// Create a new session - this may fail if the user doesn't have valid credentials.
	// In that case we just treat the version as unknown.
	sess, err := session.New(ctx, httpClient)
	if err != nil {
		return instanceVersion, err
	}

	c := client.New(httpClient, sess)

	// Query the GraphQL API for the actual version. If this fails, we also treat the
	// version as unknown.
	var query debugInfoQuery
	if err := c.Query(context.Background(), &query, nil); err != nil {
		return instanceVersion, err
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

	return instanceVersion, nil
}

// instanceTypeWarning explains why the instance type could not be determined.
// Most causes come down to not being logged in, but a proxy auth failure is
// configuration the user has to fix, so it is reported as such rather than
// being mistaken for missing credentials.
func instanceTypeWarning(err error) string {
	var proxyErr *client.ProxyAuthError
	if errors.As(err, &proxyErr) {
		return fmt.Sprintf("Warning: Unable to determine Spacelift instance type: %v. Some commands may be unavailable until the proxy auth configuration of the current profile is fixed.", err)
	}

	return "Warning: Unable to determine Spacelift instance type. Some commands may be unavailable until you authenticate with Spacelift."
}

func main() {
	ctx := context.Background()
	instanceVersion, err := getSpaceliftInstanceVersion(ctx)

	if instanceVersion.InstanceType == cmd.SpaceliftInstanceTypeUnknown {
		log.Println(instanceTypeWarning(err))
	}

	app := &cli.Command{
		Name:                  "spacectl",
		Version:               version,
		Usage:                 "Programmatic access to Spacelift GraphQL API.",
		EnableShellCompletion: true,
		Commands: append([]*cli.Command{
			profile.Command(),
			whoami.Command(),
			registry.Command(),
			versioncmd.Command(version, instanceVersion),
		}, cmd.ResolveCommands(instanceVersion, []cmd.Command{
			api.Command(),
			module.Command(),
			stack.Command(),
			provider.Command(),
			runexternaldependency.Command(),
			workerpools.Command(),
			blueprint.Command(),
			template.Command(),
			policy.Command(),
			audittrail.Command(),
			ai.Command(),
			mcp.Command(),
		})...),
	}

	if err := app.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
