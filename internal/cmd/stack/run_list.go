package stack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func runList(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)
	if err != nil {
		return err
	}

	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}
	maxResults := cliCtx.Int(flagMaxResults.Name)

	switch outputFormat {
	case cmd.OutputFormatTable:
		return listRunsTable(cliCtx.Context, stackID, maxResults)
	case cmd.OutputFormatJSON:
		return listRunsJSON(cliCtx.Context, stackID, maxResults)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

type runsJSONQuery struct {
	ID       string `graphql:"id" json:"id"`
	Branch   string `graphql:"branch" json:"branch"`
	CanRetry bool   `graphql:"canRetry" json:"canRetry"`
	Comments []struct {
		Body      string `graphql:"body" json:"body"`
		CreatedAt int    `graphql:"createdAt" json:"createdAt"`
		Username  string `graphql:"username" json:"username"`
	} `graphql:"comments" json:"comments"`
	Commit struct {
		AuthorLogin string `graphql:"authorLogin" json:"authorLogin"`
		AuthorName  string `graphql:"authorName" json:"authorName"`
		Hash        string `graphql:"hash" json:"hash"`
	} `graphql:"commit" json:"commit"`
	CreatedAt int `graphql:"createdAt" json:"createdAt"`
	Delta     *struct {
		AddCount    int `graphql:"addCount" json:"addCount"`
		ChangeCount int `graphql:"changeCount" json:"changeCount"`
		DeleteCount int `graphql:"deleteCount" json:"deleteCount"`
		Resources   int `graphql:"resources" json:"resources"`
	} `graphql:"delta" json:"delta"`
	DriftDetection bool   `graphql:"driftDetection" json:"driftDetection"`
	Expired        bool   `graphql:"expired" json:"expired"`
	IsMostRecent   bool   `graphql:"isMostRecent" json:"isMostRecent"`
	NeedsApproval  bool   `graphql:"needsApproval" json:"needsApproval"`
	State          string `graphql:"state" json:"state"`
	Title          string `graphql:"title" json:"title"`
	TriggeredBy    string `graphql:"triggeredBy" json:"triggeredBy"`
}

func listRunsJSON(ctx context.Context, stackID string, maxResults int) error {
	var results []runsJSONQuery
	var before *string

	for len(results) < maxResults {
		var query struct {
			Stack *struct {
				Runs []runsJSONQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return errors.Wrap(err, "failed to query run list")
		}

		if query.Stack == nil {
			return fmt.Errorf("stack %q not found", stackID)
		}

		if len(query.Stack.Runs) == 0 {
			break
		}

		resultsToAdd := maxResults - len(results)
		if resultsToAdd > len(query.Stack.Runs) {
			resultsToAdd = len(query.Stack.Runs)
		}

		results = append(results, query.Stack.Runs[:resultsToAdd]...)

		before = &query.Stack.Runs[len(query.Stack.Runs)-1].ID
	}

	return cmd.OutputJSON(results)
}

type runsTableQuery struct {
	ID     string `graphql:"id"`
	State  string `graphql:"state"`
	Title  string `graphql:"title"`
	Commit struct {
		AuthorName string `graphql:"authorName"`
		Hash       string `graphql:"hash"`
	} `graphql:"commit"`
	CreatedAt   int    `graphql:"createdAt"`
	TriggeredBy string `graphql:"triggeredBy"`
	Delta       struct {
		AddCount    int `graphql:"addCount"`
		ChangeCount int `graphql:"changeCount"`
		DeleteCount int `graphql:"deleteCount"`
	} `graphql:"delta"`
}

func listRunsTable(ctx context.Context, stackID string, maxResults int) error {
	var results []runsTableQuery
	var before *string

	for len(results) < maxResults {
		var query struct {
			Stack *struct {
				Runs []runsTableQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return errors.Wrap(err, "failed to query run list")
		}

		if query.Stack == nil {
			return fmt.Errorf("stack %q not found", stackID)
		}

		if len(query.Stack.Runs) == 0 {
			break
		}

		resultsToAdd := maxResults - len(results)
		if resultsToAdd > len(query.Stack.Runs) {
			resultsToAdd = len(query.Stack.Runs)
		}

		results = append(results, query.Stack.Runs[:resultsToAdd]...)

		before = &query.Stack.Runs[len(query.Stack.Runs)-1].ID
	}

	tableData := [][]string{{"ID", "Status", "Message", "Commit", "Triggered At", "Triggered By", "Changes"}}
	for _, run := range results {
		var deltaComponents []string

		if run.Delta.AddCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("+%d", run.Delta.AddCount))
		}
		if run.Delta.ChangeCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("~%d", run.Delta.ChangeCount))
		}
		if run.Delta.DeleteCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("-%d", run.Delta.DeleteCount))
		}

		delta := strings.Join(deltaComponents, " ")

		triggeredBy := run.TriggeredBy
		if triggeredBy == "" {
			triggeredBy = "Git commit"
		}

		createdAt := time.Unix(int64(run.CreatedAt), 0)

		tableData = append(tableData, []string{
			run.ID,
			run.State,
			run.Title,
			cmd.HumanizeGitHash(run.Commit.Hash),
			createdAt.Format(time.RFC3339),
			triggeredBy,
			delta,
		})
	}

	return cmd.OutputTable(tableData, true)
}
