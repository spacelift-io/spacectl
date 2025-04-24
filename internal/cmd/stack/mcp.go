package stack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func RegisterMCPTools(s *server.MCPServer) {
	registerListStacksTool(s)
	registerListStackRunsTool(s)
	registerGetStackRunLogsTool(s)
	registerGetStackRunChangesTool(s)
	registerTriggerStackRunTool(s)
	registerDiscardStackRunTool(s)
	registerConfirmStackRunTool(s)
	registerListResourcesTool(s)
}

func registerListStacksTool(s *server.MCPServer) {
	stacksTool := mcp.NewTool("list_stacks",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift stacks. Use the pagination cursor to navigate through multiple pages of results.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Stacks",
			ReadOnlyHint: true,
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of stacks to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on stack name, description, and tags")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(stacksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := 50
		if request.Params.Arguments["limit"] != nil {
			limit = int(request.Params.Arguments["limit"].(float64))
		}
		var fullTextSearch *graphql.String
		if request.Params.Arguments["search"] != nil {
			search := request.Params.Arguments["search"].(string)
			fullTextSearch = graphql.NewString(graphql.String(search))
		}

		var nextPageCursor *graphql.String
		if request.Params.Arguments["next_page_cursor"] != nil {
			cursor := request.Params.Arguments["next_page_cursor"].(string)
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
		}

		result, err := searchStacks[stack](ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search stacks")
		}

		stacksJSON, err := json.Marshal(result.Stacks)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal stacks to JSON")
		}

		output := string(stacksJSON)

		if len(result.Stacks) == 0 {
			if nextPageCursor != nil {
				output = "No more stacks available. You have reached the end of the list."
			} else {
				output = "No stacks found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d stacks:\n%s\n\nThis is not the complete list. To view more stacks, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.Stacks), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d stacks:\n%s\n\nThis is the complete list. There are no more stacks to fetch.", len(result.Stacks), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerListStackRunsTool(s *server.MCPServer) {
	stackRunsTool := mcp.NewTool("list_stack_runs",
		mcp.WithDescription(`Retrieve a paginated list of runs for a specific Spacelift stack Use the pagination cursor to navigate through the run history.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Stack Runs",
			ReadOnlyHint: true,
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack to list runs for"), mcp.Required()),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(stackRunsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)

		var before *string
		if request.Params.Arguments["next_page_cursor"] != nil {
			cursor := request.Params.Arguments["next_page_cursor"].(string)
			before = &cursor
		}

		var query struct {
			Stack *struct {
				Runs []runsJSONQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}
		if query.Stack == nil {
			return nil, errors.Errorf("failed to lookup runs for stack %q", stackID)
		}

		runs := query.Stack.Runs

		runsJSON, err := json.Marshal(runs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal runs to JSON")
		}

		output := string(runsJSON)

		if len(runs) == 0 {
			if before != nil {
				output = fmt.Sprintf("No more runs available for stack %q. You have reached the end of the list.", stackID)
			} else {
				output = fmt.Sprintf("No runs found for stack %q.", stackID)
			}
		} else {
			nextPageCursor := runs[len(runs)-1].ID
			output = fmt.Sprintf("Showing runs for stack %q:\n%s\n\nThis is not the complete list. To view more runs, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", stackID, output, nextPageCursor)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetStackRunLogsTool(s *server.MCPServer) {
	stackRunLogsTool := mcp.NewTool("get_stack_run_logs",
		mcp.WithDescription(`Retrieve the complete logs for a specific run of a Spacelift stack. Shows all output generated during the run execution, including commands, errors, and results.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Stack Run Logs",
			ReadOnlyHint: true,
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
	)

	s.AddTool(stackRunLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)
		runID := request.Params.Arguments["run_id"].(string)

		// Create a channel to collect log lines
		logLines := make(chan string)
		var allLogs []string
		var terminal *structs.RunStateTransition
		var err error

		// Start a goroutine to collect logs
		done := make(chan struct{})
		go func() {
			for line := range logLines {
				allLogs = append(allLogs, line)
			}
			close(done)
		}()

		// Run the log collection
		go func() {
			terminal, err = runStates(ctx, stackID, runID, logLines, nil)
			close(logLines)
		}()

		// Wait for log collection to complete
		<-done

		if err != nil {
			return nil, errors.Wrap(err, "failed to collect run logs")
		}

		// Format the output
		output := fmt.Sprintf("Logs for run %s in stack %s:\n\n", runID, stackID)

		for _, line := range allLogs {
			output += line
		}

		if terminal != nil {
			output += fmt.Sprintf("\n\nRun completed with state: %s", terminal.State)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetStackRunChangesTool(s *server.MCPServer) {
	stackRunChangesTool := mcp.NewTool("get_stack_run_changes",
		mcp.WithDescription(`Retrieve the resource changes detected or applied during a specific Spacelift stack run.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Stack Run Changes",
			ReadOnlyHint: true,
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
	)

	s.AddTool(stackRunChangesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)
		runID := request.Params.Arguments["run_id"].(string)

		changes, err := getRunChanges(ctx, stackID, runID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get run changes")
		}

		changesJSON, err := json.Marshal(changes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal changes to JSON")
		}

		output := string(changesJSON)
		if len(changes) == 0 {
			output = fmt.Sprintf("No changes found for run %s in stack %s", runID, stackID)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerTriggerStackRunTool(s *server.MCPServer) {
	stackRunTriggerTool := mcp.NewTool("trigger_stack_run",
		mcp.WithDescription(`Initiate a new run for a Spacelift stack. You can specify the run type (PROPOSED or TRACKED) and optionally provide a specific commit SHA to use for the run.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Trigger Stack Run",
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("commit_sha", mcp.Description("The commit SHA to use for the run")),
		mcp.WithString("run_type", mcp.Description("The type of run to trigger (default: TRACKED)"),
			mcp.Enum("PROPOSED", "TRACKED")),
	)

	s.AddTool(stackRunTriggerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)

		// Default run type is TRACKED
		runType := "TRACKED"
		if request.Params.Arguments["run_type"] != nil {
			runType = request.Params.Arguments["run_type"].(string)
		}

		var mutation struct {
			RunTrigger struct {
				ID string `graphql:"id"`
			} `graphql:"runTrigger(stack: $stack, commitSha: $sha, runType: $type)"`
		}

		variables := map[string]any{
			"stack": graphql.ID(stackID),
			"sha":   (*graphql.String)(nil),
			"type":  structs.NewRunType(runType),
		}

		if request.Params.Arguments["commit_sha"] != nil {
			commitSha := request.Params.Arguments["commit_sha"].(string)
			variables["sha"] = graphql.NewString(graphql.String(commitSha))
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return nil, errors.Wrap(err, "failed to trigger run")
		}

		output := fmt.Sprintf("Successfully created a %s\n", runType)
		output += fmt.Sprintf("The live run can be visited at %s", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunTrigger.ID,
		))

		return mcp.NewToolResultText(output), nil
	})
}

func registerDiscardStackRunTool(s *server.MCPServer) {
	stackRunDiscardTool := mcp.NewTool("discard_stack_run",
		mcp.WithDescription(`Discard a pending or in-progress run for a Spacelift stack.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Discard Stack Run",
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
	)

	s.AddTool(stackRunDiscardTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)
		runID := request.Params.Arguments["run_id"].(string)

		var mutation struct {
			RunDiscard struct {
				ID string `graphql:"id"`
			} `graphql:"runDiscard(stack: $stack, run: $run)"`
		}

		variables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(runID),
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return nil, errors.Wrap(err, "failed to discard run")
		}

		output := "You have successfully discarded a deployment\n"
		output += fmt.Sprintf("The run can be visited at %s", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunDiscard.ID,
		))

		return mcp.NewToolResultText(output), nil
	})
}

func registerConfirmStackRunTool(s *server.MCPServer) {
	stackRunConfirmTool := mcp.NewTool("confirm_stack_run",
		mcp.WithDescription(`Approve a run that is waiting for confirmation in a Spacelift stack. This allows the run to proceed with applying changes to your infrastructure.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Confirm Stack Run",
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
	)

	s.AddTool(stackRunConfirmTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.Params.Arguments["stack_id"].(string)
		runID := request.Params.Arguments["run_id"].(string)

		var mutation struct {
			RunConfirm struct {
				ID string `graphql:"id"`
			} `graphql:"runConfirm(stack: $stack, run: $run)"`
		}

		variables := map[string]any{
			"stack": graphql.ID(stackID),
			"run":   graphql.ID(runID),
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return nil, errors.Wrap(err, "failed to confirm run")
		}

		output := "You have successfully confirmed a deployment\n"
		output += fmt.Sprintf("The live run can be visited at %s", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			mutation.RunConfirm.ID,
		))

		return mcp.NewToolResultText(output), nil
	})
}

func registerListResourcesTool(s *server.MCPServer) {
	resourcesTool := mcp.NewTool("list_resources",
		mcp.WithDescription(`Retrieve a list of infrastructure resources managed by all Spacelift stacks or a specific stack.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Resources",
			ReadOnlyHint: true,
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack. If not provided, resources for all stacks will be listed")),
	)

	s.AddTool(resourcesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if stackID, ok := request.Params.Arguments["stack_id"].(string); ok && stackID != "" {
			return listResourcesForOneStack(ctx, stackID)
		}

		return listResourcesForAllStacks(ctx)
	})
}

func listResourcesForOneStack(ctx context.Context, id string) (*mcp.CallToolResult, error) {
	var query struct {
		Stack stackWithResources `graphql:"stack(id: $id)"`
	}

	variables := map[string]any{"id": graphql.ID(id)}
	if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query stack resources: %w", err)
	}

	resourcesJSON, err := json.Marshal(query.Stack)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal stack resources to JSON")
	}

	output := string(resourcesJSON)
	if len(query.Stack.Entities) == 0 {
		output = fmt.Sprintf("No resources found for stack %q", id)
	}

	return mcp.NewToolResultText(output), nil
}

func listResourcesForAllStacks(ctx context.Context) (*mcp.CallToolResult, error) {
	var query struct {
		Stacks []stackWithResources `graphql:"stacks" json:"stacks,omitempty"`
	}

	if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
		return nil, fmt.Errorf("failed to query all stacks resources: %w", err)
	}

	resourcesJSON, err := json.Marshal(query.Stacks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal all stacks resources to JSON")
	}

	output := string(resourcesJSON)
	if len(query.Stacks) == 0 {
		output = "No stacks found"
	}

	return mcp.NewToolResultText(output), nil
}
