package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

const maxLocalPreviewRunLogLines = 1000

type McpOptions struct {
	UseHeadersForLocalPreview bool
}

func RegisterMCPTools(s *server.MCPServer, options McpOptions) {
	registerListStacksTool(s)
	registerListStackRunsTool(s)
	registerListStackProposedRunsTool(s)
	registerGetStackRunTool(s)
	registerGetStackRunLogsTool(s)
	registerGetStackRunChangesTool(s)
	registerTriggerStackRunTool(s)
	registerDiscardStackRunTool(s)
	registerConfirmStackRunTool(s)
	registerListResourcesTool(s)
	registerLocalPreviewTool(s, options)
}

func registerListStacksTool(s *server.MCPServer) {
	stacksTool := mcp.NewTool("list_stacks",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift stacks. Use the pagination cursor to navigate through multiple pages of results.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Stacks",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of stacks to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on stack name, description, and tags")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(stacksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := 50
		if request.GetArguments()["limit"] != nil {
			limit = int(request.GetArguments()["limit"].(float64))
		}
		var fullTextSearch *graphql.String
		if request.GetArguments()["search"] != nil {
			search := request.GetArguments()["search"].(string)
			fullTextSearch = graphql.NewString(graphql.String(search))
		}

		var nextPageCursor *graphql.String
		if request.GetArguments()["next_page_cursor"] != nil {
			cursor := request.GetArguments()["next_page_cursor"].(string)
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
		mcp.WithDescription(`Retrieve a paginated list of tracked runs (runs making changes in resources) for a specific Spacelift stack Use the pagination cursor to navigate through the run history. This tool does not include proposed (preview) runs`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Stack Runs",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack to list runs for"), mcp.Required()),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(stackRunsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)

		var before *string
		if request.GetArguments()["next_page_cursor"] != nil {
			cursor := request.GetArguments()["next_page_cursor"].(string)
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

func registerListStackProposedRunsTool(s *server.MCPServer) {
	stackRunsTool := mcp.NewTool("list_stack_proposed_runs",
		mcp.WithDescription(`Retrieve a paginated list of preview (including local preview) runs (runs showing preview of introduced changes) for a specific Spacelift stack Use the pagination cursor to navigate through the run history. This tools does not include tracked runs`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Stack Runs",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack to list runs for"), mcp.Required()),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(stackRunsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)

		var before *string
		if request.GetArguments()["next_page_cursor"] != nil {
			cursor := request.GetArguments()["next_page_cursor"].(string)
			before = &cursor
		}

		var query struct {
			Stack *struct {
				ProposedRuns []runsJSONQuery `graphql:"proposedRuns(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]any{"stackId": stackID, "before": before}); err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}
		if query.Stack == nil {
			return nil, errors.Errorf("failed to lookup runs for stack %q", stackID)
		}

		runs := query.Stack.ProposedRuns

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

func registerGetStackRunTool(s *server.MCPServer) {
	stackRunsTool := mcp.NewTool("get_stack_run",
		mcp.WithDescription(`Retrieve a specific run for a Spacelift stack. Use the pagination cursor to navigate through the run history.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Stack Run",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack to list runs for"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run to retrieve"), mcp.Required()),
	)

	s.AddTool(stackRunsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)
		runID := request.GetArguments()["run_id"].(string)

		var query struct {
			Stack *struct {
				Run *runsJSONQuery `graphql:"run(id: $runId)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]any{"stackId": stackID, "runId": runID}); err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}
		if query.Stack == nil {
			return mcp.NewToolResultText("Stack not found"), nil
		}
		if query.Stack.Run == nil {
			return mcp.NewToolResultText("Run not found"), nil
		}

		runs := query.Stack.Run

		runJSON, err := json.Marshal(runs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal runs to JSON")
		}

		output := string(runJSON)

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetStackRunLogsTool(s *server.MCPServer) {
	stackRunLogsTool := mcp.NewTool("get_stack_run_logs",
		mcp.WithDescription(`Retrieve logs for a specific run of a Spacelift stack. Shows output generated during the run execution, including commands, errors, and results. You can use skip and limit parameters to retrieve specific line ranges.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Stack Run Logs",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
		mcp.WithNumber("skip", mcp.Description("Skip the first N lines of the log output. Can be used together with `limit` parameter for pagination.")),
		mcp.WithNumber("limit", mcp.Description("The maximum number of log lines to retrieve. Can be used together with `skip` parameter for pagination.")),
	)

	s.AddTool(stackRunLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)
		runID := request.GetArguments()["run_id"].(string)

		var skip, limit *int
		if request.GetArguments()["skip"] != nil {
			skipVal := int(request.GetArguments()["skip"].(float64))
			if skipVal >= 0 {
				skip = &skipVal
			}
		}

		if request.GetArguments()["limit"] != nil {
			limitVal := int(request.GetArguments()["limit"].(float64))
			if limitVal >= 0 {
				limit = &limitVal
			}
		}

		logLines := make(chan string)
		var allLogs []string
		var terminal *structs.RunStateTransition
		var err error

		go func() {
			for line := range logLines {
				allLogs = append(allLogs, line)
			}
		}()

		terminal, err = runStates(ctx, stackID, runID, logLines, nil)
		close(logLines)

		if err != nil {
			return nil, errors.Wrap(err, "failed to collect run logs")
		}

		output := fmt.Sprintf("Logs for run %s in stack %s:\n\n", runID, stackID)

		for _, line := range allLogs {
			output += line
		}

		if terminal != nil {
			output += fmt.Sprintf("\n\nRun completed with state: %s", terminal.State)
		}

		output = trimStringByLines(output, skip, limit)

		return mcp.NewToolResultText(output), nil
	})
}

func trimStringByLines(input string, skip *int, limit *int) string {
	startLine := 0
	if skip != nil {
		startLine = max(*skip, 0)
	}

	lines := strings.Split(input, "\n")

	if startLine >= len(lines) {
		return fmt.Sprintf("No more log lines available. You have reached the end of the list. Total number of log lines: %d. Skipped %d lines.", len(lines), startLine)
	}

	endLine := len(lines)
	if limit != nil {
		endLine = min(startLine+*limit, len(lines))
	}

	newResult := strings.Join(lines[startLine:endLine], "\n")

	if startLine > 0 {
		newResult = fmt.Sprintf("... %d lines before ...\n%s", startLine, newResult)
	}

	if endLine < len(lines) {
		newResult = fmt.Sprintf("%s\n... %d more lines available ...", newResult, len(lines)-endLine)
	}

	return newResult
}

func registerGetStackRunChangesTool(s *server.MCPServer) {
	stackRunChangesTool := mcp.NewTool("get_stack_run_changes",
		mcp.WithDescription(`Retrieve the resource changes detected or applied during a specific Spacelift stack run.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Stack Run Changes",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("The ID of the run"), mcp.Required()),
	)

	s.AddTool(stackRunChangesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)
		runID := request.GetArguments()["run_id"].(string)

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
		stackID := request.GetArguments()["stack_id"].(string)

		// Default run type is TRACKED
		runType := "TRACKED"
		if request.GetArguments()["run_type"] != nil {
			runType = request.GetArguments()["run_type"].(string)
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

		if request.GetArguments()["commit_sha"] != nil {
			commitSha := request.GetArguments()["commit_sha"].(string)
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
		stackID := request.GetArguments()["stack_id"].(string)
		runID := request.GetArguments()["run_id"].(string)

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
		stackID := request.GetArguments()["stack_id"].(string)
		runID := request.GetArguments()["run_id"].(string)

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
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack. If not provided, resources for all stacks will be listed")),
	)

	s.AddTool(resourcesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if stackID, ok := request.GetArguments()["stack_id"].(string); ok && stackID != "" {
			return listResourcesForOneStack(ctx, stackID)
		}

		return listResourcesForAllStacks(ctx)
	})
}

func registerLocalPreviewTool(s *server.MCPServer, options McpOptions) {
	var localPreviewTool = mcp.NewTool("local_preview",
		mcp.WithDescription(`Start a preview (proposed run) based on the current project.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Local Preview",
		}),
		mcp.WithString("stack_id", mcp.Description("The ID of the stack"), mcp.Required()),
		mcp.WithObject("environment_variables", mcp.Description("Environment variables to set for the run")),
		mcp.WithArray("targets", mcp.Description("Limit the planning operation to only the given module, resource, or resource instance and all of its dependencies.")),
		mcp.WithString("path", mcp.Description("The path to the local workspace. If not provided, the current working directory will be used.")),
		mcp.WithString(
			"await_for_completion",
			mcp.Description("Wait for the run to complete before returning the result. If true, the tool will wait for the run to complete and return logs once it is complete."),
			mcp.DefaultString("true"),
			mcp.Enum("true", "false"),
		),
	)

	s.AddTool(localPreviewTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stackID := request.GetArguments()["stack_id"].(string)

		stack, err := stackGetByID[stack](ctx, stackID)
		if errors.Is(err, errNoStackFound) {
			return mcp.NewToolResultText("Stack not found"), nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to check if stack exists: %w", err)
		}

		if !stack.LocalPreviewEnabled {
			linkToStack := authenticated.Client.URL("/stack/%s", stack.ID)
			return mcp.NewToolResultText(fmt.Sprintf("Local preview has not been enabled for this stack, please enable local preview in the stack settings: %s", linkToStack)), nil
		}

		var envVars []EnvironmentVariable
		if request.GetArguments()["environment_variables"] != nil {
			v := request.GetArguments()["environment_variables"].(map[string]any)
			for k, v := range v {
				if o, ok := v.(string); ok {
					envVars = append(envVars, EnvironmentVariable{
						Key:   graphql.String(k),
						Value: graphql.String(o),
					})
				}
			}
		}

		var targets []string
		if request.GetArguments()["targets"] != nil {
			v := request.GetArguments()["targets"].([]any)
			for _, t := range v {
				if o, ok := t.(string); ok {
					targets = append(targets, o)
				}
			}
		}

		var path *string
		if p, ok := request.GetArguments()["path"].(string); ok && p != "" {
			path = &p
		}

		awaitForCompletion := true
		if a, ok := request.GetArguments()["await_for_completion"].(string); ok && a != "" {
			awaitForCompletion = a == "true"
		}

		// Create a string builder to capture output
		var outputBuilder strings.Builder

		runID, err := createLocalPreviewRun(
			ctx,
			LocalPreviewOptions{
				StackID:            stackID,
				EnvironmentVars:    envVars,
				Targets:            targets,
				Path:               path,
				FindRepositoryRoot: false,
				DisregardGitignore: false,
				UseHeaders:         options.UseHeadersForLocalPreview,
				NoUpload:           false,
				ShowUploadProgress: false,
			},
			&outputBuilder,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create local preview run: %w", err)
		}

		// Get the captured output
		output := outputBuilder.String()

		// Add run URL to the output
		linkToRun := authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			runID,
		)
		output += fmt.Sprintf("\nThe live run can be visited at %s\n", linkToRun)

		if !awaitForCompletion {
			return mcp.NewToolResultText(output), nil
		}

		logLines := make(chan string)
		var allLogs []string

		go func() {
			for line := range logLines {
				allLogs = append(allLogs, line)
			}
		}()

		terminal, err := runStates(ctx, stackID, runID, logLines, nil)
		close(logLines)

		if err != nil {
			return nil, fmt.Errorf("failed to collect run logs: %w", err)
		}

		// Add logs to output
		output += "\n\nRun logs:\n\n"
		for _, line := range allLogs {
			output += line
		}

		if terminal != nil {
			output += fmt.Sprintf("\n\nRun '%s' completed with state: %s\n", runID, terminal.State)
		}

		outputLines := strings.Split(output, "\n")
		outputLinesCount := len(outputLines)
		// Limit the number of lines, because mcp clients often limit amount of ouput tokens. "get_stack_run_logs" tool has pagination and can be used to view the full logs
		if outputLinesCount > maxLocalPreviewRunLogLines {
			fromIndex := outputLinesCount - maxLocalPreviewRunLogLines

			outputLines = outputLines[fromIndex:]
			output = strings.Join(outputLines, "\n")
			output = fmt.Sprintf("... %d lines omitted. Use 'get_stack_run_logs' tool to view the full logs ...\n%s", outputLinesCount-maxLocalPreviewRunLogLines, output)
		}

		return mcp.NewToolResultText(output), nil
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
