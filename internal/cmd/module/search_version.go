package module

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func listVersions() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return listVersionsTable(cliCtx)
		case cmd.OutputFormatJSON:
			return listVersionsJSON(cliCtx)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listVersionsJSON(cliCtx *cli.Context) error {
	var query struct {
		Module struct {
			Verions []version `graphql:"versions(includeFailed: $includeFailed)"`
		} `graphql:"module(id: $id)"`
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{
		"id":            cliCtx.String(flagModuleID.Name),
		"includeFailed": graphql.Boolean(false),
	}); err != nil {
		return errors.Wrap(err, "failed to query list of modules")
	}
	return cmd.OutputJSON(query.Module.Verions)
}

func listVersionsTable(cliCtx *cli.Context) error {
	var query struct {
		Module struct {
			Verions []version `graphql:"versions(includeFailed: $includeFailed)"`
		} `graphql:"module(id: $id)"`
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{
		"id":            cliCtx.String(flagModuleID.Name),
		"includeFailed": graphql.Boolean(false),
	}); err != nil {
		return errors.Wrap(err, "failed to query list of modules")
	}

	columns := []string{"ID", "Author", "Message", "Number", "State", "Tests", "Timestamp"}
	tableData := [][]string{columns}

	if len(query.Module.Verions) > 20 {
		query.Module.Verions = query.Module.Verions[:20]
	}

	// We print the versions in reverse order
	// so the latest version is at the bottom, much easier to read in terminal.
	for i := len(query.Module.Verions) - 1; i >= 0; i-- {
		module := query.Module.Verions[i]
		row := []string{
			module.ID,
			module.Commit.AuthorName,
			module.Commit.Message,
			module.Number,
			module.State,
			fmt.Sprintf("%d", module.VersionCount),
			fmt.Sprintf("%d", module.Commit.Timestamp),
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

type version struct {
	ID     string `json:"id" graphql:"id"`
	Commit struct {
		AuthorLogin string `json:"authorLogin" graphql:"authorLogin"`
		AuthorName  string `json:"authorName" graphql:"authorName"`
		Hash        string `json:"hash" graphql:"hash"`
		Message     string `json:"message" graphql:"message"`
		Timestamp   int    `json:"timestamp" graphql:"timestamp"`
		URL         string `json:"url" graphql:"url"`
	} `json:"commit" graphql:"commit"`
	DownloadLink interface{} `json:"downloadLink" graphql:"downloadLink"`
	Number       string      `json:"number" graphql:"number"`
	SourceURL    string      `json:"sourceURL" graphql:"sourceURL"`
	State        string      `json:"state" graphql:"state"`
	VersionCount int         `json:"versionCount" graphql:"versionCount"`
	Yanked       bool        `json:"yanked" graphql:"yanked"`
}
