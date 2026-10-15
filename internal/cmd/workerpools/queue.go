package workerpools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const (
	maxSchedulableRunsLimit  = 100
	maxClientFilterScanPages = 100
)

type schedulableRunsConnection struct {
	Edges    []schedulableRunEdge `graphql:"edges"`
	PageInfo structs.PageInfo     `graphql:"pageInfo"`
}

type schedulableRunEdge struct {
	Node struct {
		StackID  string `graphql:"stackId" json:"stackId"`
		Position int    `graphql:"position" json:"position"`
		Run      struct {
			ID             string `graphql:"id" json:"id"`
			CreatedAt      int    `graphql:"createdAt" json:"createdAt"`
			State          string `graphql:"state" json:"state"`
			Type           string `graphql:"type" json:"type"`
			Title          string `graphql:"title" json:"title"`
			DriftDetection bool   `graphql:"driftDetection" json:"driftDetection"`
			IsPrioritized  bool   `graphql:"isPrioritized" json:"isPrioritized"`
			Commit         struct {
				Hash string `graphql:"hash" json:"hash"`
			} `graphql:"commit" json:"commit"`
		} `graphql:"run" json:"run"`
	} `graphql:"node" json:"node"`
}

type queueListJSON struct {
	Runs     []queueListRunJSON `json:"runs"`
	PageInfo structs.PageInfo   `json:"pageInfo"`
}

type queueListRunJSON struct {
	StackID  string               `json:"stackId"`
	Position int                  `json:"position"`
	Run      queueListRunBodyJSON `json:"run"`
}

type queueListRunBodyJSON struct {
	ID             string `json:"id"`
	CreatedAt      int    `json:"createdAt"`
	State          string `json:"state"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	DriftDetection bool   `json:"driftDetection"`
	Priority       bool   `json:"priority"`
	CommitHash     string `json:"commitHash"`
}

// schedulableClientFilter applies filters that searchSchedulableRuns does not support as predicates
// (stack id, drift, commit substring). See collectSchedulableRuns.
type schedulableClientFilter struct {
	stackID   string
	drift     *bool
	commitSub string
}

func (f *schedulableClientFilter) active() bool {
	if f == nil {
		return false
	}
	return f.stackID != "" || f.drift != nil || f.commitSub != ""
}

func (f *schedulableClientFilter) matches(edge schedulableRunEdge) bool {
	if f == nil || !f.active() {
		return true
	}
	if f.stackID != "" && !strings.EqualFold(strings.TrimSpace(edge.Node.StackID), strings.TrimSpace(f.stackID)) {
		return false
	}
	if f.drift != nil && edge.Node.Run.DriftDetection != *f.drift {
		return false
	}
	if f.commitSub != "" {
		h := strings.ToLower(edge.Node.Run.Commit.Hash)
		if !strings.Contains(h, strings.ToLower(strings.TrimSpace(f.commitSub))) {
			return false
		}
	}
	return true
}

type listQueueCommand struct{}

type discardQueueCommand struct{}

func (c *listQueueCommand) listQueue(ctx context.Context, cliCmd *cli.Command) error {
	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	base, cf, limit, err := queueSearchPlan(cliCmd)
	if err != nil {
		return err
	}

	edges, pageInfo, err := collectSchedulableRuns(ctx, cliCmd.String(flagWorkerPoolIDOptional.Name), base, cf, limit)
	if err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return c.showQueueTable(edges, pageInfo)
	case cmd.OutputFormatJSON:
		return c.showQueueJSON(edges, pageInfo)
	default:
		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func (c *listQueueCommand) showQueueJSON(edges []schedulableRunEdge, pageInfo structs.PageInfo) error {
	runs := make([]queueListRunJSON, 0, len(edges))
	for _, edge := range edges {
		r := edge.Node.Run
		runs = append(runs, queueListRunJSON{
			StackID:  edge.Node.StackID,
			Position: edge.Node.Position,
			Run: queueListRunBodyJSON{
				ID:             r.ID,
				CreatedAt:      r.CreatedAt,
				State:          r.State,
				Type:           r.Type,
				Title:          r.Title,
				DriftDetection: r.DriftDetection,
				Priority:       r.IsPrioritized,
				CommitHash:     r.Commit.Hash,
			},
		})
	}
	return cmd.OutputJSON(queueListJSON{Runs: runs, PageInfo: pageInfo})
}

func (c *listQueueCommand) showQueueTable(edges []schedulableRunEdge, pageInfo structs.PageInfo) error {
	tableData := [][]string{{"#", "Stack ID", "Run ID", "Commit", "Drift detection", "Priority", "State", "Type", "Title", "Created At"}}
	for _, edge := range edges {
		tm := time.Unix(int64(edge.Node.Run.CreatedAt), 0)
		title := edge.Node.Run.Title
		if title == "" {
			title = "-"
		}
		commit := edge.Node.Run.Commit.Hash
		if commit == "" {
			commit = "-"
		} else {
			commit = cmd.HumanizeGitHash(commit)
		}
		tableData = append(tableData, []string{
			fmt.Sprint(edge.Node.Position),
			edge.Node.StackID,
			edge.Node.Run.ID,
			commit,
			fmt.Sprintf("%t", edge.Node.Run.DriftDetection),
			fmt.Sprintf("%t", edge.Node.Run.IsPrioritized),
			edge.Node.Run.State,
			edge.Node.Run.Type,
			title,
			tm.Format(time.RFC3339),
		})
	}
	if err := cmd.OutputTable(tableData, true); err != nil {
		return err
	}
	if pageInfo.HasNextPage && pageInfo.EndCursor != "" {
		fmt.Fprintf(os.Stderr, "\nMore results available. Pass --after=%q to fetch the next page.\n", pageInfo.EndCursor)
	}
	return nil
}

func (c *discardQueueCommand) discardQueue(ctx context.Context, cliCmd *cli.Command) error {
	runID := cliCmd.String(flagQueueRun.Name)
	if runID != "" {
		stackID := cliCmd.String(flagQueueStack.Name)
		if stackID == "" {
			return fmt.Errorf("--stack is required when --run is set")
		}
		return discardSingleQueuedRun(ctx, stackID, runID)
	}

	base, cf, limit, err := queueSearchPlan(cliCmd)
	if err != nil {
		return err
	}

	edges, _, err := collectSchedulableRuns(ctx, cliCmd.String(flagWorkerPoolIDOptional.Name), base, cf, limit)
	if err != nil {
		return err
	}

	if len(edges) == 0 {
		fmt.Println("No matching runs in the queue.")
		return nil
	}

	if !cliCmd.Bool(flagQueueYes.Name) {
		return fmt.Errorf("refusing to discard %d run(s) without --yes", len(edges))
	}

	for _, edge := range edges {
		discardedID, err := mutateRunDiscard(ctx, edge.Node.StackID, edge.Node.Run.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to discard run %q for stack %q", edge.Node.Run.ID, edge.Node.StackID)
		}
		fmt.Printf("Discarded run %s\n", discardedID)
		fmt.Println("The run can be visited at", authenticated.Client().URL(
			"/stack/%s/run/%s",
			edge.Node.StackID,
			discardedID,
		))
	}

	return nil
}

func discardSingleQueuedRun(ctx context.Context, stackID, runID string) error {
	discardedID, err := mutateRunDiscard(ctx, stackID, runID)
	if err != nil {
		return err
	}

	fmt.Println("You have successfully discarded a deployment")
	fmt.Println("The run can be visited at", authenticated.Client().URL(
		"/stack/%s/run/%s",
		stackID,
		discardedID,
	))

	return nil
}

func mutateRunDiscard(ctx context.Context, stackID, runID string) (string, error) {
	var mutation struct {
		RunDiscard struct {
			ID string `graphql:"id"`
		} `graphql:"runDiscard(stack: $stack, run: $run)"`
	}

	variables := map[string]any{
		"stack": graphql.ID(stackID),
		"run":   graphql.ID(runID),
	}

	if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
		return "", err
	}

	return mutation.RunDiscard.ID, nil
}

func collectSchedulableRuns(
	ctx context.Context,
	workerPoolID string,
	base structs.SearchInput,
	cf *schedulableClientFilter,
	limit int,
) ([]schedulableRunEdge, structs.PageInfo, error) {
	if cf == nil || !cf.active() {
		inp := base
		inp.First = graphql.NewInt(graphql.Int(limit)) //nolint:gosec
		return searchSchedulableRuns(ctx, workerPoolID, inp)
	}

	var out []schedulableRunEdge
	var lastPI structs.PageInfo
	after := base.After

	for page := 0; page < maxClientFilterScanPages; page++ {
		inp := base
		inp.First = graphql.NewInt(graphql.Int(maxSchedulableRunsLimit)) //nolint:gosec
		inp.After = after

		edges, pi, err := searchSchedulableRuns(ctx, workerPoolID, inp)
		if err != nil {
			return nil, structs.PageInfo{}, err
		}
		lastPI = pi

		for _, edge := range edges {
			if !cf.matches(edge) {
				continue
			}
			out = append(out, edge)
			if len(out) >= limit {
				return out, lastPI, nil
			}
		}

		if !pi.HasNextPage || pi.EndCursor == "" {
			break
		}
		s := graphql.String(pi.EndCursor)
		after = &s
	}

	return out, lastPI, nil
}

func searchSchedulableRuns(ctx context.Context, workerPoolID string, input structs.SearchInput) ([]schedulableRunEdge, structs.PageInfo, error) {
	if workerPoolID != "" {
		var query struct {
			WorkerPool *struct {
				Runs schedulableRunsConnection `graphql:"searchSchedulableRuns(input: $input)"`
			} `graphql:"workerPool(id: $id)"`
		}

		variables := map[string]any{
			"id":    graphql.ID(workerPoolID),
			"input": input,
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, structs.PageInfo{}, errors.Wrap(err, "failed to query schedulable runs")
		}

		if query.WorkerPool == nil {
			return nil, structs.PageInfo{}, fmt.Errorf("worker pool with id %q not found", workerPoolID)
		}

		return query.WorkerPool.Runs.Edges, query.WorkerPool.Runs.PageInfo, nil
	}

	var query struct {
		WorkerPool struct {
			Runs schedulableRunsConnection `graphql:"searchSchedulableRuns(input: $input)"`
		} `graphql:"publicWorkerPool"`
	}

	variables := map[string]any{"input": input}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return nil, structs.PageInfo{}, errors.Wrap(err, "failed to query schedulable runs")
	}

	return query.WorkerPool.Runs.Edges, query.WorkerPool.Runs.PageInfo, nil
}

func queueSearchPlan(cliCmd *cli.Command) (structs.SearchInput, *schedulableClientFilter, int, error) {
	limit := cliCmd.Int(flagQueueLimit.Name)
	if limit < 1 || limit > maxSchedulableRunsLimit {
		return structs.SearchInput{}, nil, 0, fmt.Errorf("limit must be between 1 and %d", maxSchedulableRunsLimit)
	}

	input := structs.SearchInput{
		OrderBy: &structs.QueryOrder{
			Field:     graphql.String("position"),
			Direction: graphql.String("ASC"),
		},
	}

	if after := cliCmd.String(flagQueueAfter.Name); after != "" {
		s := graphql.String(after)
		input.After = &s
	}

	if ft := cliCmd.String(flagQueueSearch.Name); ft != "" {
		s := graphql.String(ft)
		input.FullTextSearch = &s
	}

	preds, err := buildAPISchedulableRunPredicates(cliCmd)
	if err != nil {
		return structs.SearchInput{}, nil, 0, err
	}
	if len(preds) > 0 {
		input.Predicates = &preds
	}

	cf, err := clientFilterFromFlags(cliCmd)
	if err != nil {
		return structs.SearchInput{}, nil, 0, err
	}

	return input, cf, limit, nil
}

func clientFilterFromFlags(cliCmd *cli.Command) (*schedulableClientFilter, error) {
	f := &schedulableClientFilter{}
	if s := strings.TrimSpace(cliCmd.String(flagQueueStack.Name)); s != "" {
		f.stackID = s
	}
	if cliCmd.IsSet(flagQueueDriftDetection.Name) {
		b, err := parseQueueOptionalBool(cliCmd.String(flagQueueDriftDetection.Name))
		if err != nil {
			return nil, err
		}
		f.drift = &b
	}
	if c := strings.TrimSpace(cliCmd.String(flagQueueCommit.Name)); c != "" {
		f.commitSub = c
	}
	return f, nil
}

// buildAPISchedulableRunPredicates returns only predicates supported by searchSchedulableRuns.
// Stack, drift, and commit are applied client-side (see schedulableClientFilter).
func buildAPISchedulableRunPredicates(cliCmd *cli.Command) ([]structs.QueryPredicate, error) {
	var preds []structs.QueryPredicate

	if cliCmd.IsSet(flagQueuePrioritized.Name) {
		wantPrioritized, err := parseQueueOptionalBool(cliCmd.String(flagQueuePrioritized.Name))
		if err != nil {
			return nil, err
		}
		preds = append(preds, structs.QueryPredicate{
			Field: graphql.String("isPrioritized"),
			Constraint: structs.QueryFieldConstraint{
				BooleanEquals: &[]graphql.Boolean{graphql.Boolean(wantPrioritized)},
			},
		})
	}

	states := cliCmd.StringSlice(flagQueueState.Name)
	if len(states) > 0 {
		enums := make([]graphql.String, 0, len(states))
		for _, st := range states {
			st = strings.TrimSpace(st)
			if st != "" {
				enums = append(enums, graphql.String(strings.ToUpper(st)))
			}
		}
		if len(enums) > 0 {
			preds = append(preds, structs.QueryPredicate{
				Field: graphql.String("state"),
				Constraint: structs.QueryFieldConstraint{
					EnumEquals: &enums,
				},
			})
		}
	}

	for _, st := range cliCmd.StringSlice(flagQueueExcludeState.Name) {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		enum := graphql.String(strings.ToUpper(st))
		preds = append(preds, structs.QueryPredicate{
			Field:   graphql.String("state"),
			Exclude: true,
			Constraint: structs.QueryFieldConstraint{
				EnumEquals: &[]graphql.String{enum},
			},
		})
	}

	runTypes := cliCmd.StringSlice(flagQueueRunType.Name)
	if len(runTypes) > 0 {
		enums := make([]graphql.String, 0, len(runTypes))
		for _, rt := range runTypes {
			rt = strings.TrimSpace(rt)
			if rt != "" {
				enums = append(enums, graphql.String(strings.ToUpper(rt)))
			}
		}
		if len(enums) > 0 {
			preds = append(preds, structs.QueryPredicate{
				Field: graphql.String("type"),
				Constraint: structs.QueryFieldConstraint{
					EnumEquals: &enums,
				},
			})
		}
	}

	return preds, nil
}

func parseQueueOptionalBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q (use true or false)", s)
	}
}
