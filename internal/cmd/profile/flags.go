package profile

import "github.com/urfave/cli/v2"

var flagBindPort = &cli.IntFlag{
	Name:     "bind",
	Usage:    "[Optional] specify the port used for binding the server when logging in through a browser",
	Required: false,
	EnvVars:  []string{"SPACECTL_BIND_PORT"},
}
