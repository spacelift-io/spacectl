package data

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// WorkerPool allows to interact with a worker pool.
type WorkerPool struct {
	WokerPoolID string
}

// Selected opens the selected worker pool in the browser.
func (q *WorkerPool) Selected(row table.Row) error {
	return browser.OpenURL(authenticated.Client.URL("/stack/%s/run/%s", row[1], row[2]))
}

// Columns returns the columns of the worker pool table.
func (q *WorkerPool) Columns() []table.Column {
	return []table.Column{
		{Title: "#", Width: 2},
		{Title: "Stack", Width: 25},
		{Title: "Run", Width: 32},
		{Title: "State", Width: 15},
		{Title: "Type", Width: 10},
		{Title: "Created At", Width: 27},
	}
}

// Rows returns the rows of the worker pool table.
func (q *WorkerPool) Rows(ctx context.Context) (rows []table.Row, err error) {
	var runs []runsEdge
	if q.WokerPoolID == "" {
		runs, err = q.getPublicPoolRuns(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		runs, err = q.getPrivatePoolRuns(ctx)
		if err != nil {
			return nil, err
		}
	}

	for _, edge := range runs {
		tm := time.Unix(int64(edge.Node.Run.CreatedAt), 0)
		rows = append(rows, table.Row{
			fmt.Sprint(edge.Node.Position),
			edge.Node.StackID,
			edge.Node.Run.ID,
			edge.Node.Run.State,
			edge.Node.Run.Type,
			tm.Format(time.DateTime),
		})
	}

	return rows, nil
}

func (q *WorkerPool) getPublicPoolRuns(ctx context.Context) ([]runsEdge, error) {
	var query struct {
		WorkerPool struct {
			Runs runsQuery `graphql:"searchSchedulableRuns(input: $input)"`
		} `graphql:"publicWorkerPool"`
	}

	if err := authenticated.Client.Query(ctx, &query, q.baseSearchParams()); err != nil {
		return nil, errors.Wrap(err, "failed to query run list")
	}

	return query.WorkerPool.Runs.Edges, nil
}

func (q *WorkerPool) getPrivatePoolRuns(ctx context.Context) ([]runsEdge, error) {
	var query struct {
		WorkerPool struct {
			Runs runsQuery `graphql:"searchSchedulableRuns(input: $input)"`
		} `graphql:"workerPool(id: $id)"`
	}

	vars := q.baseSearchParams()
	vars["id"] = q.WokerPoolID

	if err := authenticated.Client.Query(ctx, &query, vars); err != nil {
		return nil, errors.Wrap(err, "failed to query run list")
	}

	return query.WorkerPool.Runs.Edges, nil
}

func (q *WorkerPool) baseSearchParams() map[string]interface{} {
	return map[string]interface{}{
		"input": structs.SearchInput{
			First: internal.Ptr(100),
			OrderBy: &structs.QueryOrder{
				Field:     "position",
				Direction: "ASC",
			},
		},
	}
}

type runsQuery struct {
	Edges    []runsEdge       `graphql:"edges"`
	PageInfo structs.PageInfo `graphql:"pageInfo"`
}

type runsEdge struct {
	Node struct {
		StackID  string `graphql:"stackId"`
		Run      run    `graphql:"run"`
		Position int    `graphql:"position"`
	} `graphql:"node"`
}

type run struct {
	ID        string `graphql:"id" json:"id"`
	CreatedAt int    `graphql:"createdAt" json:"createdAt"`
	State     string `graphql:"state" json:"state"`
	Type      string `graphql:"type" json:"type"`
	Title     string `graphql:"title" json:"title"`
}
