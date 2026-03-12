package workerpool

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
	registerListWorkerPoolsTool(s)
	registerGetWorkerPoolTool(s)
}

type workerPoolNode struct {
	ID          string  `graphql:"id" json:"id"`
	Name        string  `graphql:"name" json:"name"`
	Description *string `graphql:"description" json:"description"`
	CreatedAt   int     `graphql:"createdAt" json:"createdAt"`
	UpdatedAt   int     `graphql:"updatedAt" json:"updatedAt"`
	Deleted     bool    `graphql:"deleted" json:"deleted"`
	Space       struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	Labels       []string `graphql:"labels" json:"labels"`
	WorkersCount int      `graphql:"workersCount" json:"workersCount"`
	BusyWorkers  int      `graphql:"busyWorkers" json:"busyWorkers"`
	PendingRuns  int      `graphql:"pendingRuns" json:"pendingRuns"`
}

type workerPoolDetail struct {
	ID          string  `graphql:"id" json:"id"`
	Name        string  `graphql:"name" json:"name"`
	Description *string `graphql:"description" json:"description"`
	CreatedAt   int     `graphql:"createdAt" json:"createdAt"`
	UpdatedAt   int     `graphql:"updatedAt" json:"updatedAt"`
	Deleted     bool    `graphql:"deleted" json:"deleted"`
	Space       struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	} `graphql:"spaceDetails" json:"spaceDetails"`
	Labels              []string `graphql:"labels" json:"labels"`
	WorkersCount        int      `graphql:"workersCount" json:"workersCount"`
	BusyWorkers         int      `graphql:"busyWorkers" json:"busyWorkers"`
	PendingRuns         int      `graphql:"pendingRuns" json:"pendingRuns"`
	ManagedByK8sController bool  `graphql:"managedByK8sController" json:"managedByK8sController"`
}

type searchWorkerPoolsResult struct {
	WorkerPools []workerPoolNode
	PageInfo    structs.PageInfo
}

func searchWorkerPools(ctx context.Context, input structs.SearchInput) (*searchWorkerPoolsResult, error) {
	var query struct {
		SearchWorkerPoolsOutput struct {
			Edges []struct {
				Node workerPoolNode `graphql:"node"`
			} `graphql:"edges"`
			PageInfo structs.PageInfo `graphql:"pageInfo"`
		} `graphql:"searchWorkerPools(input: $input)"`
	}

	variables := map[string]any{"input": input}

	if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
		return nil, errors.Wrap(err, "failed to execute worker pools search query")
	}

	nodes := make([]workerPoolNode, 0, len(query.SearchWorkerPoolsOutput.Edges))
	for _, edge := range query.SearchWorkerPoolsOutput.Edges {
		nodes = append(nodes, edge.Node)
	}

	return &searchWorkerPoolsResult{
		WorkerPools: nodes,
		PageInfo:    query.SearchWorkerPoolsOutput.PageInfo,
	}, nil
}

func registerListWorkerPoolsTool(s *server.MCPServer) {
	tool := mcp.NewTool("list_worker_pools",
		mcp.WithDescription(`Retrieve a paginated list of Spacelift worker pools. Worker pools are groups of workers that execute stack runs.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Worker Pools",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithNumber("limit", mcp.Description("The maximum number of worker pools to return, default is 50")),
		mcp.WithString("search", mcp.Description("Perform a full text search on worker pool name and description")),
		mcp.WithString("next_page_cursor", mcp.Description("The pagination cursor to use for fetching the next page of results")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		limit := request.GetInt("limit", 50)

		var fullTextSearch *graphql.String
		if searchParam := request.GetString("search", ""); searchParam != "" {
			fullTextSearch = graphql.NewString(graphql.String(searchParam))
		}

		var nextPageCursor *graphql.String
		if cursor := request.GetString("next_page_cursor", ""); cursor != "" {
			nextPageCursor = graphql.NewString(graphql.String(cursor))
		}

		pageInput := structs.SearchInput{
			First:          graphql.NewInt(graphql.Int(limit)), //nolint: gosec
			FullTextSearch: fullTextSearch,
			After:          nextPageCursor,
		}

		result, err := searchWorkerPools(ctx, pageInput)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search worker pools")
		}

		poolsJSON, err := json.Marshal(result.WorkerPools)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal worker pools to JSON")
		}

		output := string(poolsJSON)

		if len(result.WorkerPools) == 0 {
			if nextPageCursor != nil {
				output = "No more worker pools available. You have reached the end of the list."
			} else {
				output = "No worker pools found matching your criteria."
			}
		} else if result.PageInfo.HasNextPage {
			output = fmt.Sprintf("Showing %d worker pools:\n%s\n\nThis is not the complete list. To view more worker pools, use this tool again with this cursor as \"next_page_cursor\": \"%s\"", len(result.WorkerPools), output, result.PageInfo.EndCursor)
		} else {
			output = fmt.Sprintf("Showing %d worker pools:\n%s\n\nThis is the complete list. There are no more worker pools to fetch.", len(result.WorkerPools), output)
		}

		return mcp.NewToolResultText(output), nil
	})
}

func registerGetWorkerPoolTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_worker_pool",
		mcp.WithDescription(`Retrieve detailed information about a specific Spacelift worker pool including worker counts and status.`),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Worker Pool",
			ReadOnlyHint: mcp.ToBoolPtr(true),
		}),
		mcp.WithString("worker_pool_id", mcp.Description("The ID of the worker pool to retrieve"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authenticated.Ensure(ctx, nil)
		poolID, err := request.RequireString("worker_pool_id")
		if err != nil {
			return nil, err
		}

		var query struct {
			WorkerPool *workerPoolDetail `graphql:"workerPool(id: $workerPoolId)"`
		}

		variables := map[string]any{
			"workerPoolId": graphql.ID(poolID),
		}

		if err := authenticated.Client().Query(ctx, &query, variables); err != nil {
			return nil, errors.Wrapf(err, "failed to query for worker pool ID %q", poolID)
		}

		if query.WorkerPool == nil {
			return mcp.NewToolResultText("Worker pool not found"), nil
		}

		poolJSON, err := json.Marshal(query.WorkerPool)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal worker pool to JSON")
		}

		return mcp.NewToolResultText(string(poolJSON)), nil
	})
}
