package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

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

func main() {
	cmd := &cli.Command{
		Name:                  "spacectl",
		Version:               version,
		Usage:                 "Programmatic access to Spacelift GraphQL API.",
		EnableShellCompletion: true,
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
			blueprint.Command(),
			policy.Command(),
			audittrail.Command(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
