package stack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/draw"
	"github.com/urfave/cli/v2"
)

func watch(cliCtx *cli.Context) error {
	input := structs.SearchInput{
		OrderBy: &structs.QueryOrder{
			Field:     "starred",
			Direction: "DESC",
		},
	}

	st := &Stacks{input}
	t, err := draw.NewTable(cliCtx.Context, st)
	if err != nil {
		return err
	}

	return t.DrawTable()
}

type Stacks struct {
	si structs.SearchInput
}

// Selected opens the selected worker pool in the browser.
func (q *Stacks) Filtered(s string) error {
	fullTextSearch := graphql.NewString(graphql.String(s))
	q.si.FullTextSearch = fullTextSearch

	return nil
}

// Selected opens the selected worker pool in the browser.
func (q *Stacks) Selected(row table.Row) error {
	ctx := context.Background()
	st := &StackWatch{row[0]}
	t, err := draw.NewTable(ctx, st)
	if err != nil {
		return err
	}

	return t.DrawTable()
}

// Columns returns the columns of the worker pool table.
func (q *Stacks) Columns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 25},
		{Title: "Name", Width: 25},
		{Title: "Commit", Width: 15},
		{Title: "Author", Width: 15},
		{Title: "State", Width: 15},
		{Title: "Worker Pool", Width: 15},
		{Title: "Locked By", Width: 15},
	}
}

// Rows returns the rows of the worker pool table.
func (q *Stacks) Rows(ctx context.Context) (rows []table.Row, err error) {
	stacks, err := searchAllStacks(ctx, q.si)
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks {
		rows = append(rows, table.Row{
			stack.ID,
			stack.Name,
			cmd.HumanizeGitHash(stack.TrackedCommit.Hash),
			stack.TrackedCommit.AuthorName,
			stack.State,
			stack.WorkerPool.ID,
			stack.LockedBy,
		})
	}

	return rows, nil
}

type StackWatch struct {
	id string
}

func (w *StackWatch) Filtered(s string) error {
	return nil
}

func (s *StackWatch) Columns() []table.Column {
	return []table.Column{
		{Title: "Run ID", Width: 25},
		{Title: "Status", Width: 25},
		{Title: "Message", Width: 15},
		{Title: "Commit", Width: 15},
		{Title: "Triggered At", Width: 15},
		{Title: "Triggered By", Width: 15},
		{Title: "Changes", Width: 15},
		{Title: "Stack ID", Width: 15},
	}
}

func (s *StackWatch) Rows(ctx context.Context) (rows []table.Row, err error) {
	// TODO: make maxResults configurable
	runs, err := listRuns(ctx, s.id, 100)
	if err != nil {
		return nil, err
	}

	rows = append(rows, table.Row{"<-back", "", "", "", "", "", ""})
	for _, run := range runs {
		rows = append(rows, table.Row{
			run[0],
			run[1],
			run[2],
			run[3],
			run[4],
			run[5],
			run[6],
			run[7],
		})
	}

	return rows, nil
}

func (s *StackWatch) Selected(row table.Row) error {
	ctx := context.Background()
	if row[0] == "<-back" {
		input := structs.SearchInput{
			OrderBy: &structs.QueryOrder{
				Field:     "starred",
				Direction: "DESC",
			},
		}

		st := &Stacks{input}
		t, err := draw.NewTable(ctx, st)
		if err != nil {
			return err
		}

		return t.DrawTable()
	}

	return browser.OpenURL(authenticated.Client.URL("/stack/%s/run/%s", row[7], row[0]))
}

func listRuns(ctx context.Context, stackID string, maxResults int) ([][]string, error) {
	var results []runsTableQuery
	var before *string

	for len(results) < maxResults {
		var query struct {
			Stack *struct {
				Runs []runsTableQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}

		if query.Stack == nil {
			return nil, fmt.Errorf("stack %q not found", stackID)
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

	var tableData [][]string
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
			stackID,
		})
	}

	return tableData, nil
}
