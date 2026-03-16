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
)

func taskList(ctx context.Context, cliCmd *cli.Command) error {
	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	stackID, err := getStackID(ctx, cliCmd)
	if err != nil {
		return err
	}
	maxResults := cliCmd.Int(flagMaxResults.Name)

	switch outputFormat {
	case cmd.OutputFormatTable:
		return listTasksTable(ctx, stackID, maxResults)
	case cmd.OutputFormatJSON:
		return listTasksJSON(ctx, stackID, maxResults)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

type tasksJSONQuery struct {
	ID      string `graphql:"id" json:"id"`
	Command string `graphql:"command" json:"command"`
	Commit  struct {
		Hash string `graphql:"hash" json:"hash"`
		URL  string `graphql:"url" json:"url"`
	} `graphql:"commit" json:"commit"`
	CreatedAt   int    `graphql:"createdAt" json:"createdAt"`
	State       string `graphql:"state" json:"state"`
	TriggeredBy string `graphql:"triggeredBy" json:"triggeredBy"`
	Type        string `graphql:"type" json:"type"`
}

func (t tasksJSONQuery) Cursor() string {
	return t.ID
}

type tasksTableQuery struct {
	ID      string `graphql:"id"`
	Command string `graphql:"command"`
	Commit  struct {
		Hash string `graphql:"hash"`
	} `graphql:"commit"`
	CreatedAt   int    `graphql:"createdAt"`
	State       string `graphql:"state"`
	TriggeredBy string `graphql:"triggeredBy"`
}

func (t tasksTableQuery) Cursor() string {
	return t.ID
}

func queryTasks[T any](ctx context.Context, stackID string, before *string) ([]T, error) {
	var query struct {
		Stack *struct {
			Tasks []T `graphql:"tasks(before: $before)"`
		} `graphql:"stack(id: $stackId)"`
	}

	if err := authenticated.Client().Query(ctx, &query, map[string]any{"stackId": stackID, "before": before}); err != nil {
		return nil, errors.Wrap(err, "failed to query task list")
	}

	if query.Stack == nil {
		return nil, fmt.Errorf("stack %q not found", stackID)
	}

	return query.Stack.Tasks, nil
}

func listTasksJSON(ctx context.Context, stackID string, maxResults int) error {
	results, err := fetchRuns(ctx, maxResults, func(ctx context.Context, before *string) ([]tasksJSONQuery, error) {
		return queryTasks[tasksJSONQuery](ctx, stackID, before)
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(results)
}

func listTasksTable(ctx context.Context, stackID string, maxResults int) error {
	results, err := fetchRuns(ctx, maxResults, func(ctx context.Context, before *string) ([]tasksTableQuery, error) {
		return queryTasks[tasksTableQuery](ctx, stackID, before)
	})
	if err != nil {
		return err
	}

	tableData := [][]string{{"ID", "Status", "Command", "Commit", "Triggered At", "Triggered By"}}
	for _, task := range results {
		triggeredBy := task.TriggeredBy
		if triggeredBy == "" {
			triggeredBy = "Git commit"
		}

		createdAt := time.Unix(int64(task.CreatedAt), 0)

		tableData = append(tableData, []string{
			task.ID,
			task.State,
			truncateCommand(task.Command),
			cmd.HumanizeGitHash(task.Commit.Hash),
			createdAt.Format(time.RFC3339),
			triggeredBy,
		})
	}

	return cmd.OutputTable(tableData, true)
}

func truncateCommand(command string) string {
	parts := strings.Fields(command)
	result := strings.Join(parts, " ")
	if len(result) > 60 {
		return result[:57] + "..."
	}
	return result
}
