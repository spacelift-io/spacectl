package workerpools

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

type worker struct {
	ID       string `graphql:"id" json:"id"`
	Busy     bool   `graphql:"busy" json:"busy"`
	Drained  bool   `graphql:"drained" json:"drained"`
	Metadata string `graphql:"metadata" json:"metadata"`
}

type listWorkersQuery struct {
	Pool *struct {
		Workers []worker `graphql:"workers" json:"workers"`
	} `graphql:"workerPool(id: $workerPool)"`
}

type listWorkersCommand struct{}

func (c *listWorkersCommand) listWorkers(cliCtx *cli.Context) error {
	outputFormat, err := cmd.GetOutputFormat(cliCtx)

	if err != nil {
		return err
	}

	workerPoolID := cliCtx.String(flagPoolIDNamed.Name)

	var query listWorkersQuery
	variables := map[string]interface{}{
		"workerPool": workerPoolID,
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return err
	}

	if query.Pool == nil {
		return errors.New(fmt.Sprintf("workerpool with id %s not found", workerPoolID))
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showOutputsTable(query.Pool.Workers)
	case cmd.OutputFormatJSON:
		return c.showOutputsJSON(query.Pool.Workers)
	default:
		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func (c *listWorkersCommand) showOutputsJSON(workers []worker) error {
	var output []interface{}
	for _, worker := range workers {
		var parsedMetadata map[string]interface{}

		if err := json.Unmarshal([]byte(worker.Metadata), &parsedMetadata); err != nil {
			return errors.Wrapf(err, "failed to parse metadata of worker with id %s", worker.ID)
		}

		row := map[string]interface{}{
			"id":       worker.ID,
			"busy":     worker.Busy,
			"drained":  worker.Drained,
			"metadata": parsedMetadata,
		}
		output = append(output, row)
	}
	return cmd.OutputJSON(output)
}

func (c *listWorkersCommand) showOutputsTable(workers []worker) error {
	tableData := [][]string{{"ID", "Busy", "Drained"}}
	for _, worker := range workers {
		row := []string{
			worker.ID,
			fmt.Sprintf("%v", worker.Busy),
			fmt.Sprintf("%v", worker.Drained),
		}
		tableData = append(tableData, row)
	}
	return cmd.OutputTable(tableData, true)
}
