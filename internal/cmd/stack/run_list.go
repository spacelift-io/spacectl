package stack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/nullable"
)

func runList(ctx context.Context, cliCmd *cli.Command) error {
	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	stackID, err := getStackID(cliCmd)
	if err != nil {
		return err
	}
	maxResults := cliCmd.Int(flagMaxResults.Name)
	showPreview := cliCmd.Bool(flagPreviewRuns.Name)

	switch outputFormat {
	case cmd.OutputFormatTable:
		return listRunsTable(ctx, maxResults, func(ctx context.Context, before *string) ([]runsTableQuery, error) {
			if showPreview {
				return queryPreviewRuns[runsTableQuery](ctx, stackID, before)
			}
			return queryTrackedRuns[runsTableQuery](ctx, stackID, before)
		})
	case cmd.OutputFormatJSON:
		return listRunsJSON(ctx, maxResults, func(ctx context.Context, before *string) ([]runsJSONQuery, error) {
			if showPreview {
				return queryPreviewRuns[runsJSONQuery](ctx, stackID, before)
			}
			return queryTrackedRuns[runsJSONQuery](ctx, stackID, before)
		})
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

func (r runsJSONQuery) Cursor() string {
	return r.ID
}

type withCursor interface {
	Cursor() string
}

func queryTrackedRuns[T any](ctx context.Context, stackID string, before *string) ([]T, error) {
	var query struct {
		Stack *struct {
			Runs []T `graphql:"runs(before: $before)"`
		} `graphql:"stack(id: $stackId)"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
		return nil, errors.Wrap(err, "failed to query run list")
	}

	if query.Stack == nil {
		return nil, fmt.Errorf("stack %q not found", stackID)
	}

	return query.Stack.Runs, nil
}

func queryPreviewRuns[T any](ctx context.Context, stackID string, before *string) ([]T, error) {
	var query struct {
		Stack *struct {
			ProposedRuns []T `graphql:"proposedRuns(before: $before)"`
		} `graphql:"stack(id: $stackId)"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
		return nil, errors.Wrap(err, "failed to query run list")
	}

	if query.Stack == nil {
		return nil, fmt.Errorf("stack %q not found", stackID)
	}

	return query.Stack.ProposedRuns, nil

}

func fetchRuns[T withCursor](ctx context.Context, maxResults int, fetcher func(context.Context, *string) ([]T, error)) ([]T, error) {
	var results []T
	var before *string

	for len(results) < maxResults {
		runs, err := fetcher(ctx, before)

		if err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}

		if len(runs) == 0 {
			break
		}

		resultsToAdd := min(maxResults-len(results), len(runs))

		results = append(results, runs[:resultsToAdd]...)

		before = nullable.OfValue(runs[len(runs)-1].Cursor())
	}

	return results, nil
}

func listRunsJSON(ctx context.Context, maxResults int, fetcher func(context.Context, *string) ([]runsJSONQuery, error)) error {
	results, err := fetchRuns(ctx, maxResults, fetcher)
	if err != nil {
		return err
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

func (r runsTableQuery) Cursor() string {
	return r.ID
}

func listRunsTable(ctx context.Context, maxResults int, fetcher func(context.Context, *string) ([]runsTableQuery, error)) error {
	results, err := fetchRuns(ctx, maxResults, fetcher)
	if err != nil {
		return err
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
