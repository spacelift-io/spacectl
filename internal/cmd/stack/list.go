package stack

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/urfave/cli/v2"
)

func listStacks() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		outputFormat, err := cmd.GetOutputFormat(cliCtx)
		if err != nil {
			return err
		}

		var limit *uint
		if cliCtx.IsSet(cmd.FlagLimit.Name) {
			if cliCtx.Uint(cmd.FlagLimit.Name) == 0 {
				return fmt.Errorf("limit must be greater than 0")
			}

			if cliCtx.Uint(cmd.FlagLimit.Name) >= math.MaxInt32 {
				return fmt.Errorf("limit must be less than %d", math.MaxInt32)
			}

			limit = internal.Ptr(cliCtx.Uint(cmd.FlagLimit.Name))
		}

		var search *string
		if cliCtx.IsSet(cmd.FlagSearch.Name) {
			if cliCtx.String(cmd.FlagSearch.Name) == "" {
				return fmt.Errorf("search must be non-empty")
			}

			search = internal.Ptr(cliCtx.String(cmd.FlagSearch.Name))
		}

		switch outputFormat {
		case cmd.OutputFormatTable:
			return listStacksTable(cliCtx, search, limit)
		case cmd.OutputFormatJSON:
			return listStacksJSON(cliCtx, search, limit)
		}

		return fmt.Errorf("unknown output format: %v", outputFormat)
	}
}

func listStacksJSON(
	ctx *cli.Context,
	search *string,
	limit *uint,
) error {
	var first *int
	if limit != nil {
		first = internal.Ptr(int(*limit))
	}

	stacks, err := searchAllStacks(ctx.Context, structs.SearchInput{
		First:          first,
		FullTextSearch: search,
	})
	if err != nil {
		return err
	}

	return cmd.OutputJSON(stacks)
}

func listStacksTable(
	ctx *cli.Context,
	search *string,
	limit *uint,
) error {
	var first *int
	if limit != nil {
		first = internal.Ptr(int(*limit))
	}

	input := structs.SearchInput{
		First:          first,
		FullTextSearch: search,
		OrderBy: &structs.QueryOrder{
			Field:     "starred",
			Direction: "DESC",
		},
	}

	stacks, err := searchAllStacks(ctx.Context, input)
	if err != nil {
		return err
	}

	columns := []string{"Name", "ID", "Commit", "Author", "State", "Worker Pool", "Locked By"}
	if ctx.Bool(cmd.FlagShowLabels.Name) {
		columns = append(columns, "Labels")
	}

	tableData := [][]string{columns}
	for _, s := range stacks {
		row := []string{
			s.Name,
			s.ID,
			cmd.HumanizeGitHash(s.TrackedCommit.Hash),
			s.TrackedCommit.AuthorName,
			s.State,
			s.WorkerPool.Name,
			s.LockedBy,
		}
		if ctx.Bool(cmd.FlagShowLabels.Name) {
			row = append(row, strings.Join(s.Labels, ", "))
		}

		tableData = append(tableData, row)
	}

	return cmd.OutputTable(tableData, true)
}

// searchStacks returns a list of stacks based on the provided search input.
// input.First limits the total number of returned stacks, if not provided all stacks are returned.
func searchAllStacks(ctx context.Context, input structs.SearchInput) ([]stack, error) {
	const maxPageSize = 50

	var limit int
	if input.First != nil {
		limit = int(*input.First)
	}
	fetchAll := limit == 0

	out := []stack{}
	pageInput := structs.SearchInput{
		First:          internal.Ptr(maxPageSize),
		FullTextSearch: input.FullTextSearch,
	}
	for {
		if !fetchAll {
			// Fetch exactly the number of items requested
			pageInput.First = internal.Ptr(slices.Min([]int{maxPageSize, limit - len(out)}))
		}

		result, err := searchStacks(ctx, pageInput)
		if err != nil {
			return nil, err
		}

		out = append(out, result.Stacks...)

		if result.PageInfo.HasNextPage && (fetchAll || limit > len(out)) {
			pageInput.After = internal.Ptr(result.PageInfo.EndCursor)
		} else {
			break
		}
	}

	return out, nil
}

type stack struct {
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
	SpaceDetails        struct {
		ID          string  `graphql:"id" json:"id,omitempty"`
		Name        string  `graphql:"name" json:"name,omitempty"`
		Description string  `graphql:"description" json:"description,omitempty"`
		ParentSpace *string `graphql:"parentSpace" json:"parentSpace,omitempty"`
	} `graphql:"spaceDetails" json:"spaceDetails,omitempty"`
	TrackedCommit struct {
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
}
