package stack

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func listStacks() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return listStacksTable(cliCtx.Context)
		case cmd.OutputFormatJSON:
			return listStacksJSON(cliCtx.Context)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listStacksJSON(ctx context.Context) error {
	var query struct {
		Stacks []struct {
			ID             string `graphql:"id" json:"id,omitempty"`
			Administrative bool   `graphql:"administrative" json:"administrative,omitempty"`
			Autodeploy     bool   `graphql:"autodeploy" json:"autodeploy,omitempty"`
			Autoretry      bool   `graphql:"autoretry" json:"autoretry,omitempty"`
			Blocker        struct {
				ID string `graphql:"id" json:"id,omitempty"`
			} `graphql:"blocker"`
			AfterApply          []string `graphql:"afterApply" json:"afterApply,omitempty"`
			BeforeApply         []string `graphql:"beforeApply" json:"beforeApply,omitempty"`
			AfterInit           []string `graphql:"afterInit" json:"afterInit,omitempty"`
			BeforeInit          []string `graphql:"beforeInit" json:"beforeInit,omitempty"`
			AfterPlan           []string `graphql:"afterPlan" json:"afterPlan,omitempty"`
			BeforePlan          []string `graphql:"beforePlan" json:"beforePlan,omitempty"`
			AfterPerform        []string `graphql:"afterPerform" json:"afterPerform,omitempty"`
			BeforePerform       []string `graphql:"beforePerform" json:"beforePerform,omitempty"`
			AfterDestroy        []string `graphql:"afterDestroy" json:"afterDestroy,omitempty"`
			BeforeDestroy       []string `graphql:"beforeDestroy" json:"beforeDestroy,omitempty"`
			Branch              string   `graphql:"branch" json:"branch,omitempty"`
			CanWrite            bool     `graphql:"canWrite" json:"canWrite,omitempty"`
			CreatedAt           int64    `graphql:"createdAt" json:"createdAt,omitempty"`
			Deleted             bool     `graphql:"deleted" json:"deleted,omitempty"`
			Deleting            bool     `graphql:"deleting" json:"deleting,omitempty"`
			Description         string   `graphql:"description" json:"description,omitempty"`
			Labels              []string `graphql:"labels" json:"labels,omitempty"`
			LocalPreviewEnabled bool     `graphql:"localPreviewEnabled" json:"localPreviewEnabled,omitempty"`
			LockedBy            string   `graphql:"lockedBy" json:"lockedBy,omitempty"`
			ManagesStateFile    bool     `graphql:"managesStateFile" json:"managesStateFile,omitempty"`
			Name                string   `graphql:"name" json:"name,omitempty"`
			Namespace           string   `graphql:"namespace" json:"namespace,omitempty"`
			ProjectRoot         string   `graphql:"projectRoot" json:"projectRoot,omitempty"`
			Provider            string   `graphql:"provider" json:"provider,omitempty"`
			Repository          string   `graphql:"repository" json:"repository,omitempty"`
			RunnerImage         string   `graphql:"runnerImage" json:"runnerImage,omitempty"`
			Starred             bool     `graphql:"starred" json:"starred,omitempty"`
			State               string   `graphql:"state" json:"state,omitempty"`
			StateSetAt          int64    `graphql:"stateSetAt" json:"stateSetAt,omitempty"`
			TerraformVersion    string   `graphql:"terraformVersion" json:"terraformVersion,omitempty"`
			TrackedCommit       struct {
				AuthorLogin string `graphql:"authorLogin" json:"authorLogin,omitempty"`
				AuthorName  string `graphql:"authorName" json:"authorName,omitempty"`
				Hash        string `graphql:"hash" json:"hash,omitempty"`
				Message     string `graphql:"message" json:"message,omitempty"`
				Timestamp   int64  `graphql:"timestamp" json:"timestamp,omitempty"`
				URL         string `graphql:"url" json:"url,omitempty"`
			} `graphql:"trackedCommit" json:"trackedCommit,omitempty"`
			TrackedCommitSetBy string `graphql:"trackedCommitSetBy" json:"trackedCommitSetBy,omitempty"`
			WorkerPool         struct {
				ID   string `graphql:"id" json:"id,omitempty"`
				Name string `graphql:"name" json:"name,omitempty"`
			} `graphql:"workerPool" json:"workerPool,omitempty"`
		} `graphql:"stacks" json:"stacks,omitempty"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "failed to query list of stacks")
	}
	return cmd.OutputJSON(query.Stacks)
}

func listStacksTable(ctx context.Context) error {
	var query struct {
		Stacks []struct {
			LockedBy      string `graphql:"lockedBy"`
			Name          string `graphql:"name"`
			State         string `graphql:"state"`
			TrackedCommit struct {
				AuthorName string `graphql:"authorName"`
				Hash       string `graphql:"hash"`
			} `graphql:"trackedCommit"`
			WorkerPool struct {
				Name string `graphql:"name"`
			} `graphql:"workerPool"`
		} `graphql:"stacks"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "failed to query list of stacks")
	}

	sort.SliceStable(query.Stacks, func(i, j int) bool {
		return strings.Compare(strings.ToLower(query.Stacks[i].Name), strings.ToLower(query.Stacks[j].Name)) < 0
	})

	tableData := [][]string{{"Name", "Commit", "Author", "State", "Worker Pool", "Locked By"}}
	for _, stack := range query.Stacks {
		var shortenedHash string
		if len(stack.TrackedCommit.Hash) > 8 {
			shortenedHash = stack.TrackedCommit.Hash[0:8]
		}

		tableData = append(tableData, []string{
			stack.Name,
			shortenedHash,
			stack.TrackedCommit.AuthorName,
			stack.State,
			stack.WorkerPool.Name,
			stack.LockedBy,
		})
	}

	return cmd.OutputTable(tableData)
}
