package profile

import "github.com/urfave/cli/v2"

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
