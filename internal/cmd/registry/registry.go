package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const (
	formatEnv         = "env"
	formatTerraformRC = "terraformrc"
)

var flagFormat = &cli.StringFlag{
	Name:  "format",
	Usage: fmt.Sprintf("Output `format`. Allowed values: %s, %s", formatEnv, formatTerraformRC),
	Value: formatEnv,
}

var flagHost = &cli.StringSliceFlag{
	Name:  "host",
	Usage: "[Optional] Registry `host` to print credentials for, replacing the hosts derived from the Spacelift endpoint. Can be repeated.",
}

// Command returns the registry commands for use in the CLI.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "registry",
		Usage: "Work with Spacelift's private module and provider registry",
		Commands: []*cli.Command{
			{
				Name: "credentials",
				Usage: "Print Terraform/OpenTofu credentials for Spacelift's private module and provider registry. " +
					"The output contains a secret token, so don't log it.",
				Flags:     []cli.Flag{flagFormat, flagHost},
				ArgsUsage: cmd.EmptyArgsUsage,
				Action:    credentials,
			},
		},
	}
}

func credentials(ctx context.Context, cliCmd *cli.Command) error {
	format := cliCmd.String(flagFormat.Name)
	if format != formatEnv && format != formatTerraformRC {
		return fmt.Errorf("unsupported format %q, allowed values: %s, %s", format, formatEnv, formatTerraformRC)
	}

	manager, err := session.UserProfileManager()
	if err != nil {
		return fmt.Errorf("could not access profile manager: %w", err)
	}

	httpClient, err := authenticated.HTTPClient()
	if err != nil {
		return err
	}

	sess, err := session.FromProfileOrEnvironment(ctx, manager, httpClient, os.LookupEnv)
	if err != nil {
		return fmt.Errorf("could not get session: %w", err)
	}

	hosts := cliCmd.StringSlice(flagHost.Name)
	if len(hosts) == 0 {
		selfHosted, err := isSelfHosted(ctx, httpClient, sess)
		if err != nil {
			return err
		}

		if hosts, err = registryHosts(sess.Endpoint(), selfHosted); err != nil {
			return err
		}
	}

	token, err := sess.BearerToken(ctx)
	if err != nil {
		return fmt.Errorf("could not get bearer token: %w", err)
	}

	return render(cliCmd.Root().Writer, format, hosts, token)
}

func isSelfHosted(ctx context.Context, httpClient *http.Client, sess session.Session) (bool, error) {
	var query struct {
		DebugInfo struct {
			SelfHostedVersion string `graphql:"selfHostedVersion"`
		} `graphql:"debugInfo"`
	}

	if err := client.New(httpClient, sess).Query(ctx, &query, nil); err != nil {
		return false, fmt.Errorf("could not tell whether the Spacelift instance is self-hosted, pass --%s to set the registry hosts: %w", flagHost.Name, err)
	}

	return query.DebugInfo.SelfHostedVersion != "", nil
}

// registryHosts follows the backend's runner rules so Terraform finds credentials
// under the hosts that serve the registry. For SaaS, acme.app.spacelift.io gives
// app.spacelift.io and spacelift.io; a domain without "app." gives one host.
// Self-hosted instances use the endpoint host. Non-default ports stay because
// Terraform matches credentials by host and port; port 443 is omitted.
func registryHosts(endpoint string, selfHosted bool) ([]string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("could not derive the registry host from endpoint %q; pass --%s", endpoint, flagHost.Name)
	}

	host := strings.ToLower(u.Host)
	if u.Port() == "443" {
		host = strings.ToLower(u.Hostname())
	}

	if selfHosted {
		return []string{host}, nil
	}

	_, domain, ok := strings.Cut(host, ".")
	if !ok || !strings.Contains(domain, ".") {
		return nil, fmt.Errorf("could not derive the registry host from endpoint %q; pass --%s", endpoint, flagHost.Name)
	}

	if bare := strings.TrimPrefix(domain, "app."); bare != domain {
		return []string{domain, bare}, nil
	}

	return []string{domain}, nil
}

// tfTokenEnvName uses Terraform and OpenTofu's hostname encoding so they can
// find the token in the environment.
func tfTokenEnvName(host string) string {
	name := strings.ReplaceAll(strings.ToLower(host), "-", "__")
	return "TF_TOKEN_" + strings.ReplaceAll(name, ".", "_")
}

func render(w io.Writer, format string, hosts []string, token string) error {
	if format == formatEnv {
		for _, host := range hosts {
			if strings.Contains(host, ":") {
				return fmt.Errorf("registry host %q has a port, which TF_TOKEN_* variables can't express; use --%s %s", host, flagFormat.Name, formatTerraformRC)
			}
		}
	}

	for _, host := range hosts {
		var err error
		if format == formatEnv {
			_, err = fmt.Fprintf(w, "%s=%s\n", tfTokenEnvName(host), token)
		} else {
			_, err = fmt.Fprintf(w, "credentials %q {\n  token = %q\n}\n", host, token)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
