package ai

import (
	"context"
	"fmt"
	"math"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

// maxSessionsPageSize is the largest perPage the backend honours.
const maxSessionsPageSize = 100

type chatSession struct {
	ID              string  `json:"id" graphql:"id"`
	Description     *string `json:"description" graphql:"description"`
	CreatedAt       int     `json:"createdAt" graphql:"createdAt"`
	UpdatedAt       int     `json:"updatedAt" graphql:"updatedAt"`
	Version         int     `json:"version" graphql:"version"`
	LastUsedAIModel *string `json:"lastUsedAIModel" graphql:"lastUsedAIModel"`
}

func listSessions(ctx context.Context, cliCmd *cli.Command) error {
	outputFormat, err := cmd.GetOutputFormat(cliCmd)
	if err != nil {
		return err
	}

	limit := cliCmd.Uint(cmd.FlagLimit.Name)
	if limit >= math.MaxInt32 {
		return fmt.Errorf("limit must be less than %d", math.MaxInt32)
	}

	var search *string
	if cliCmd.IsSet(cmd.FlagSearch.Name) {
		if cliCmd.String(cmd.FlagSearch.Name) == "" {
			return fmt.Errorf("search must be non-empty")
		}

		search = new(cliCmd.String(cmd.FlagSearch.Name))
	}

	sessions, err := fetchSessions(ctx, authenticated.Client(), search, limit)
	if err != nil {
		return err
	}

	switch outputFormat {
	case cmd.OutputFormatTable:
		return cmd.OutputTable(sessionsTable(sessions), true)
	case cmd.OutputFormatJSON:
		return cmd.OutputJSON(sessions)
	}

	return fmt.Errorf("unknown output format: %v", outputFormat)
}

// fetchSessions pages through intentChatConversations until the backend runs
// out of results or limit is reached. A limit of 0 fetches everything.
func fetchSessions(ctx context.Context, c client.Client, search *string, limit uint) ([]chatSession, error) {
	var fullTextSearch *graphql.String
	if search != nil {
		fullTextSearch = new(graphql.String(*search))
	}

	// The API paginates by page number, so perPage has to stay constant across
	// requests; any overshoot on the last page is trimmed below.
	perPage := graphql.Int(maxSessionsPageSize)
	if limit > 0 && limit < maxSessionsPageSize {
		perPage = graphql.Int(limit)
	}

	out := []chatSession{}
	for page := graphql.Int(1); ; page++ {
		var query struct {
			IntentChatConversations []chatSession `graphql:"intentChatConversations(page: $page, perPage: $perPage, search: $search)"`
		}

		variables := map[string]any{
			"page":    page,
			"perPage": perPage,
			"search":  fullTextSearch,
		}

		if err := c.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to list Infra Assistant sessions: %w", err)
		}

		out = append(out, query.IntentChatConversations...)

		if limit > 0 && uint(len(out)) >= limit {
			return out[:limit], nil
		}

		if len(query.IntentChatConversations) < int(perPage) {
			return out, nil
		}
	}
}

func sessionsTable(sessions []chatSession) [][]string {
	tableData := [][]string{{"ID", "Title", "Updated At", "Model"}}
	for _, s := range sessions {
		tableData = append(tableData, []string{
			s.ID,
			valueOrEmpty(s.Description),
			cmd.HumanizeUnixSeconds(s.UpdatedAt),
			valueOrEmpty(s.LastUsedAIModel),
		})
	}

	return tableData
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
