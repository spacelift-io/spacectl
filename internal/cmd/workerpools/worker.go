package workerpools

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

const (
	drainWorkerPollInterval = time.Second * 2
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

type drainWorkerMutation struct {
	Worker struct {
		ID string `graphql:"id" json:"id"`
	} `graphql:"workerDrainSet(workerPool: $workerPool, id: $worker, drain: $drain)"`
}

type cycleWorkerMutation struct {
	WorkerPoolCycle bool `graphql:"workerPoolCycle(id: $workerPoolId)"`
}

type cycleWorkersCommand struct{}

type listWorkersCommand struct{}

type drainWorkerCommand struct{}

type undrainWorkerCommand struct{}

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
		return fmt.Errorf("workerpool with id %s not found", workerPoolID)
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

func (c *drainWorkerCommand) drainWorker(cliCtx *cli.Context) error {
	workerID := cliCtx.String(flagWorkerID.Name)
	workerPoolID := cliCtx.String(flagPoolIDNamed.Name)
	waitUntilDrained := cliCtx.Bool(flagWaitUntilDrained.Name)

	var mutation drainWorkerMutation
	variables := map[string]interface{}{
		"worker":     graphql.ID(workerID),
		"workerPool": graphql.ID(workerPoolID),
		"drain":      true,
	}

	if err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables); err != nil {
		return err
	}

	if waitUntilDrained {
		if err := c.waitUntilDrained(cliCtx, workerID, workerPoolID); err != nil {
			return err
		}
	}

	fmt.Printf("Successfully drained worker %s\n", workerID)

	return nil
}

func (c *drainWorkerCommand) waitUntilDrained(cliCtx *cli.Context, workerID string, workerPoolID string) error {
	var workerDrainedAndIdle = false
	var firstRun = true

	for !workerDrainedAndIdle {
		if !firstRun {
			time.Sleep(drainWorkerPollInterval)
		}

		var err error
		workerDrainedAndIdle, err = c.drainedWorkerIsIdle(cliCtx, workerID, workerPoolID)

		if err != nil {
			return err
		}

		firstRun = false
	}

	return nil
}

func (c *drainWorkerCommand) drainedWorkerIsIdle(cliCtx *cli.Context, workerID string, workerPoolID string) (bool, error) {
	type worker struct {
		ID      string `graphql:"id"`
		Drained bool   `graphql:"drained"`
		Busy    bool   `graphql:"busy"`
	}

	var query struct {
		WorkerPool struct {
			Workers []worker `graphql:"workers"`
		} `graphql:"workerPool(id: $workerPool)"`
	}

	variables := map[string]interface{}{
		"workerPool": graphql.ID(workerPoolID),
	}

	if err := authenticated.Client.Query(cliCtx.Context, &query, variables); err != nil {
		return false, err
	}

	var workerToDrain *worker

	for i, w := range query.WorkerPool.Workers {
		if w.ID == workerID {
			workerToDrain = &query.WorkerPool.Workers[i]
			break
		}
	}

	if workerToDrain == nil {
		return false, errors.New("worker to drain doesn't exist anymore")
	}

	if !workerToDrain.Drained {
		return false, errors.New("worker is no longer flagged as drained")
	}

	return !workerToDrain.Busy, nil
}

func (c *undrainWorkerCommand) undrainWorker(cliCtx *cli.Context) error {
	workerID := cliCtx.String(flagWorkerID.Name)
	workerPoolID := cliCtx.String(flagPoolIDNamed.Name)

	var mutation drainWorkerMutation
	variables := map[string]interface{}{
		"worker":     graphql.ID(workerID),
		"workerPool": graphql.ID(workerPoolID),
		"drain":      false,
	}

	err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)

	if err != nil {
		return err
	}

	fmt.Printf("Successfully undrained worker %s\n", workerID)

	return nil
}

func (c *cycleWorkersCommand) cycleWorkers(cliCtx *cli.Context) error {
	var mutation cycleWorkerMutation
	variables := map[string]interface{}{
		"workerPoolId": graphql.ID(cliCtx.String(flagPoolIDNamed.Name)),
	}

	err := authenticated.Client.Mutate(cliCtx.Context, &mutation, variables)

	if err != nil {
		return err
	}

	fmt.Printf("Successfully cycled worker pool %s\n", cliCtx.String(flagPoolIDNamed.Name))

	return nil
}
