package profile

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
)

var bindHost string
var flagBindHost = &cli.StringFlag{
	Name:        "bind-host",
	Usage:       "[Optional] specify the host used for binding the server when logging in through a browser",
	Required:    false,
	Value:       "localhost",
	Destination: &bindHost,
	EnvVars:     []string{"SPACECTL_BIND_HOST"},
}

var bindPort int
var flagBindPort = &cli.IntFlag{
	Name:        "bind",
	Usage:       "[Optional] specify the port used for binding the server when logging in through a browser",
	Required:    false,
	Value:       0,
	Destination: &bindPort,
	EnvVars:     []string{"SPACECTL_BIND_PORT"},
}

const (
	methodBrowser = "browser"
	methodAPI     = "api"
	methodGithub  = "github"
)

var methodToCredentialsType = map[string]session.CredentialsType{
	methodGithub:  session.CredentialsTypeGitHubToken,
	methodAPI:     session.CredentialsTypeAPIKey,
	methodBrowser: session.CredentialsTypeAPIToken,
}

var flagMethod = &cli.StringFlag{
	Name:     "method",
	Usage:    "[Optional] the method to use for logging in to Spacelift",
	Required: false,
	EnvVars:  []string{"SPACECTL_LOGIN_METHOD"},
	Action: func(ctx *cli.Context, v string) error {
		if v == "" {
			return nil
		}

		switch v {
		case methodBrowser, methodAPI, methodGithub:
			return nil
		default:
			return fmt.Errorf("flag 'method' was provided an invalid value, possible values: %s, %s %s", methodBrowser, methodAPI, methodGithub)
		}
	},
}

var flagEndpoint = &cli.StringFlag{
	Name:     "endpoint",
	Usage:    "[Optional] the endpoint to use for logging in to Spacelift",
	Required: false,
	EnvVars:  []string{"SPACECTL_LOGIN_ENDPOINT"},
}
