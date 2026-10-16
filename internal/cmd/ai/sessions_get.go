package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type transcriptRange struct {
	offset int32
	limit  int32
}

func getSession(ctx context.Context, cliCmd *cli.Command) error {
	if cliCmd.NArg() != 1 {
		return errors.New("exactly one argument is required: the session ID")
	}

	sessionID := strings.TrimSpace(cliCmd.Args().First())
	if sessionID == "" {
		return errors.New("session ID must be non-empty")
	}

	offset := cliCmd.Int(flagTranscriptOffset.Name)
	if offset < 0 {
		return fmt.Errorf("offset must be non-negative, got %d", offset)
	}
	if offset > math.MaxInt32 {
		return fmt.Errorf("offset must be at most %d, got %d", math.MaxInt32, offset)
	}

	limit := cliCmd.Int(flagTranscriptLimit.Name)
	if limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", limit)
	}
	if limit > math.MaxInt32 {
		return fmt.Errorf("limit must be at most %d, got %d", math.MaxInt32, limit)
	}

	return writeTranscript(ctx, authenticated.Client(), sessionID, transcriptRange{offset: int32(offset), limit: int32(limit)}, os.Stdout)
}

// writeTranscript fetches the markdown transcript of a session and writes it
// to w unchanged, so it can be piped.
func writeTranscript(ctx context.Context, c client.Client, sessionID string, r transcriptRange, w io.Writer) error {
	var query struct {
		IntentChatConversation *struct {
			Transcript string `graphql:"transcript(offset: $offset, limit: $limit)"`
		} `graphql:"intentChatConversation(id: $id)"`
	}

	// The backend treats offset 0 as the first message and limit 0 as all
	// messages, the same as omitting them.
	variables := map[string]any{
		"id":     graphql.ID(sessionID),
		"offset": graphql.Int(r.offset),
		"limit":  graphql.Int(r.limit),
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return fmt.Errorf("failed to get Infra Assistant session: %w", err)
	}

	if query.IntentChatConversation == nil {
		return fmt.Errorf("session %q not found (sessions belong to the user who created them; API key profiles can't read them)", sessionID)
	}

	if _, err := io.WriteString(w, query.IntentChatConversation.Transcript); err != nil {
		return fmt.Errorf("failed to write the transcript: %w", err)
	}

	return nil
}
