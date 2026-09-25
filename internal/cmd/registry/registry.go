package registry

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

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

// Command encapsulates the registry command subtree.
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
		if hosts, err = registryHosts(sess.Endpoint()); err != nil {
			return err
		}
	}

	token, err := sess.BearerToken(ctx)
	if err != nil {
		return fmt.Errorf("could not get bearer token: %w", err)
	}

	return render(cliCmd.Root().Writer, format, hosts, token)
}

// registryHosts returns the registry hosts for a Spacelift endpoint. On SaaS the
// registry answers on the domain the account subdomain sits under and on that
// domain without "app.", the same pair runners get credentials for:
// acme.app.spacelift.io gives app.spacelift.io and spacelift.io. Self-hosted
// installations have no account subdomain, so the endpoint host is the registry.
func registryHosts(endpoint string) ([]string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("could not derive the registry host from endpoint %q; pass --%s", endpoint, flagHost.Name)
	}

	host := strings.ToLower(u.Hostname())
	labels := strings.Split(host, ".")
	for i := 1; i < len(labels)-1; i++ {
		if labels[i] == "app" {
			return []string{strings.Join(labels[i:], "."), strings.Join(labels[i+1:], ".")}, nil
		}
	}

	return []string{host}, nil
}

// tfTokenEnvName returns the TF_TOKEN_* variable Terraform and OpenTofu read a
// host's credentials from: dots become underscores and dashes double underscores.
func tfTokenEnvName(host string) string {
	name := strings.ReplaceAll(strings.ToLower(host), "-", "__")
	return "TF_TOKEN_" + strings.ReplaceAll(name, ".", "_")
}

func render(w io.Writer, format string, hosts []string, token string) error {
	for _, host := range hosts {
		var err error
		switch format {
		case formatEnv:
			_, err = fmt.Fprintf(w, "%s=%s\n", tfTokenEnvName(host), token)
		case formatTerraformRC:
			_, err = fmt.Fprintf(w, "credentials %q {\n  token = %q\n}\n", host, token)
		default:
			err = fmt.Errorf("unsupported format %q", format)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
