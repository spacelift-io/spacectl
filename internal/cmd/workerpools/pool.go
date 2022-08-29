package workerpools

import (
	"fmt"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type pool struct {
	ID          string  `graphql:"id" json:"id"`
	Name        string  `graphql:"name" json:"name"`
	Description *string `graphql:"description" json:"description"`
	PendingRuns int     `graphql:"pendingRuns" json:"pendingRuns"`
	BusyWorkers int     `graphql:"busyWorkers" json:"busyWorkers"`
	Workers     []struct {
		ID string `graphql:"id" json:"id"`
	} `graphql:"workers" json:"workers"`
}

type poolJSONOutput struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	PendingRuns       int     `json:"pendingRuns"`
	BusyWorkers       int     `json:"busyWorkers"`
	RegisteredWorkers int     `json:"registeredWorkers"`
}

type listPoolsQuery struct {
	Pools []pool `graphql:"workerPools" json:"workerPools"`
}

type listPoolsCommand struct{}

func (c *listPoolsCommand) listPools(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)

	if err != nil {
		return err
	}

	var query listPoolsQuery

	if err := authenticated.Client.Query(cliCtx.Context, &query, map[string]interface{}{}); err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showOutputsTable(query.Pools)
	case cmd.OutputFormatJSON:
		return c.showOutputsJSON(query.Pools)
	default:
		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func (c *listPoolsCommand) showOutputsTable(pools []pool) error {
	tableData := [][]string{{"ID", "Name", "Description", "Pending Runs", "Busy Workers", "Registered Workers"}}

	for _, pool := range pools {
		var row []string

		row = append(row, pool.ID)
		row = append(row, pool.Name)

		if pool.Description != nil {
			row = append(row, *pool.Description)
		} else {
			row = append(row, "")
		}

		row = append(row, fmt.Sprintf("%d", pool.PendingRuns))
		row = append(row, fmt.Sprintf("%d", pool.BusyWorkers))
		row = append(row, fmt.Sprintf("%d", len(pool.Workers)))

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

func (c *listPoolsCommand) showOutputsJSON(pools []pool) error {
	var output []poolJSONOutput
	for _, pool := range pools {
		row := pool.toJSONOutput()
		output = append(output, row)
	}
	return cmd.OutputJSON(output)
}

func (p *pool) toJSONOutput() poolJSONOutput {
	return poolJSONOutput{
		ID:                p.ID,
		Name:              p.Name,
		Description:       p.Description,
		PendingRuns:       p.PendingRuns,
		BusyWorkers:       p.BusyWorkers,
		RegisteredWorkers: len(p.Workers),
	}
}
