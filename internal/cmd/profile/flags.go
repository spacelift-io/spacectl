package profile

import (
	"fmt"
	"time"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/urfave/cli/v2"
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

const (
	usageViewCSVTimeFormat   = "2006-01-02"
	usageViewCSVDefaultRange = time.Duration(-1*30*24) * time.Hour
)

var flagUsageViewCSVSince = &cli.StringFlag{
	Name:     "since",
	Usage:    "[Optional] the start of the time range to query for usage data in format YYYY-MM-DD",
	Required: false,
	EnvVars:  []string{"SPACECTL_USAGE_VIEW_CSV_SINCE"},
	Value:    time.Now().Add(usageViewCSVDefaultRange).Format(usageViewCSVTimeFormat),
	Action: func(context *cli.Context, s string) error {
		_, err := time.Parse(usageViewCSVTimeFormat, s)
		if err != nil {
			return err
		}
		return nil
	},
}

var flagUsageViewCSVUntil = &cli.StringFlag{
	Name:     "until",
	Usage:    "[Optional] the end of the time range to query for usage data in format YYYY-MM-DD",
	Required: false,
	EnvVars:  []string{"SPACECTL_USAGE_VIEW_CSV_UNTIL"},
	Value:    time.Now().Format(usageViewCSVTimeFormat),
	Action: func(context *cli.Context, s string) error {
		_, err := time.Parse(usageViewCSVTimeFormat, s)
		if err != nil {
			return err
		}
		return nil
	},
}

const (
	aspectRunMinutes  = "run-minutes"
	aspectWorkerCount = "worker-count"
)

var aspects = map[string]struct{}{
	aspectRunMinutes:  {},
	aspectWorkerCount: {},
}

var flagUsageViewCSVAspect = &cli.StringFlag{
	Name:     "aspect",
	Usage:    "[Optional] the aspect to query for usage data",
	Required: false,
	EnvVars:  []string{"SPACECTL_USAGE_VIEW_CSV_ASPECT"},
	Value:    aspectWorkerCount,
	Action: func(context *cli.Context, s string) error {
		if _, isValidAspect := aspects[s]; !isValidAspect {
			return fmt.Errorf("invalid aspect: %s", s)
		}
		return nil
	},
}

const (
	groupByRunState = "run-state"
	groupByRunType  = "run-type"
)

var groupBys = map[string]struct{}{
	groupByRunState: {},
	groupByRunType:  {},
}

var flagUsageViewCSVGroupBy = &cli.StringFlag{
	Name:     "group-by",
	Usage:    "[Optional] the aspect to group run minutes by",
	Required: false,
	EnvVars:  []string{"SPACECTL_USAGE_VIEW_CSV_GROUP_BY"},
	Value:    groupByRunType,
	Action: func(context *cli.Context, s string) error {
		if _, isValidGroupBy := groupBys[s]; !isValidGroupBy {
			return fmt.Errorf("invalid group-by: %s", s)
		}
		return nil
	},
}

var flagUsageViewCSVFile = &cli.StringFlag{
	Name:     "file",
	Usage:    "[Optional] the file to save the CSV to",
	Required: false,
	EnvVars:  []string{"SPACECTL_USAGE_VIEW_CSV_FILE"},
}
