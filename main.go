package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/module"
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
var loggingLevel = new(slog.LevelVar)

func main() {

	compileTime, err := time.Parse(time.RFC3339, date)

	if err != nil {
		slog.Error("Could not parse compilation date", "err", err)
		os.Exit(1)
	}
	app := &cli.App{
		Name:     "spacectl",
		Version:  version,
		Compiled: compileTime,
		Usage:    "Programmatic access to Spacelift GraphQL API.",
		Commands: []*cli.Command{
			module.Command(),
			profile.Command(),
			provider.Command(),
			runexternaldependency.Command(),
			stack.Command(),
			whoami.Command(),
			versioncmd.Command(version),
			workerpools.Command(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-format",
				Aliases: []string{"f"},
				Usage:   "Log format: json, text",
				Value:   "text",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Log level: debug, info, warn, error",
				Value:   "info",
			},
		},
		Before: func(c *cli.Context) error {
			var logger *slog.Logger
			switch c.String("log-format") {
			case "json":
				logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
					AddSource: true,
					Level:     loggingLevel,
				}))
			case "text":
				logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					AddSource: true,
					Level:     loggingLevel,
				}))
			default:
				return cli.Exit("Invalid log format", 1)
			}

			switch c.String("log-level") {
			case "debug":
				loggingLevel.Set(slog.LevelDebug)
			case "info":
				loggingLevel.Set(slog.LevelInfo)
			case "warn":
				loggingLevel.Set(slog.LevelWarn)
			case "error":
				loggingLevel.Set(slog.LevelError)
			default:
				return cli.Exit("Invalid log level", 1)
			}

			slog.SetDefault(logger)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("error", "err", err)
		os.Exit(1)
	}
}
