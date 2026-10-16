package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"
)

const testToken = "test-token"

type fakeSession struct {
	endpoint string
}

func (f *fakeSession) BearerToken(context.Context) (string, error) { return testToken, nil }
func (f *fakeSession) Endpoint() string                            { return f.endpoint }
func (f *fakeSession) Type() session.CredentialsType               { return session.CredentialsTypeAPIToken }

func newTestClient(t *testing.T, handler http.Handler) client.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return client.New(srv.Client(), &fakeSession{endpoint: srv.URL + "/graphql"})
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// fakeConversationsAPI mimics the backend's page-number pagination over a fixed
// set of conversations and records the variables of every request.
type fakeConversationsAPI struct {
	t        *testing.T
	total    int
	requests []graphqlRequest
}

func (f *fakeConversationsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assert.Equal(f.t, "/graphql", r.URL.Path)
	assert.Equal(f.t, "Bearer "+testToken, r.Header.Get("Authorization"))

	var req graphqlRequest
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&req))
	f.requests = append(f.requests, req)

	page := int(req.Variables["page"].(float64))
	perPage := int(req.Variables["perPage"].(float64))

	items := []map[string]any{}
	for i := (page - 1) * perPage; i < min(page*perPage, f.total); i++ {
		item := map[string]any{
			"id":              fmt.Sprintf("conv-%03d", i),
			"description":     fmt.Sprintf("Session %d", i),
			"createdAt":       1700000000 + i,
			"updatedAt":       1700001000 + i,
			"version":         i,
			"lastUsedAIModel": "claude-sonnet",
		}
		if i == 0 {
			item["description"] = nil
			item["lastUsedAIModel"] = nil
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	require.NoError(f.t, json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"intentChatConversations": items},
	}))
}

func TestFetchSessions(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		limit        uint
		wantCount    int
		wantPerPages []float64
	}{
		{name: "empty", total: 0, limit: 0, wantCount: 0, wantPerPages: []float64{100}},
		{name: "single partial page", total: 7, limit: 0, wantCount: 7, wantPerPages: []float64{100}},
		{name: "fetches all pages", total: 250, limit: 0, wantCount: 250, wantPerPages: []float64{100, 100, 100}},
		{name: "exact page boundary", total: 200, limit: 0, wantCount: 200, wantPerPages: []float64{100, 100, 100}},
		{name: "limit below page size", total: 250, limit: 30, wantCount: 30, wantPerPages: []float64{30}},
		{name: "limit spanning pages keeps page size constant", total: 250, limit: 150, wantCount: 150, wantPerPages: []float64{100, 100}},
		{name: "limit above total", total: 5, limit: 50, wantCount: 5, wantPerPages: []float64{50}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeConversationsAPI{t: t, total: tt.total}
			c := newTestClient(t, api)

			sessions, err := fetchSessions(t.Context(), c, nil, tt.limit)
			require.NoError(t, err)
			require.Len(t, sessions, tt.wantCount)

			for i, s := range sessions {
				assert.Equal(t, fmt.Sprintf("conv-%03d", i), s.ID)
			}

			gotPerPages := make([]float64, 0, len(api.requests))
			for i, req := range api.requests {
				assert.InDelta(t, float64(i+1), req.Variables["page"], 0)
				assert.Nil(t, req.Variables["search"])
				gotPerPages = append(gotPerPages, req.Variables["perPage"].(float64))
			}
			assert.Equal(t, tt.wantPerPages, gotPerPages)
		})
	}
}

func TestFetchSessions_PassesSearchAndDecodesFields(t *testing.T) {
	api := &fakeConversationsAPI{t: t, total: 2}
	c := newTestClient(t, api)

	sessions, err := fetchSessions(t.Context(), c, new("terraform"), 0)
	require.NoError(t, err)

	require.Len(t, api.requests, 1)
	assert.Equal(t, "terraform", api.requests[0].Variables["search"])
	assert.Contains(t, api.requests[0].Query, "intentChatConversations(page: $page, perPage: $perPage, search: $search)")

	require.Len(t, sessions, 2)
	assert.Nil(t, sessions[0].Description)
	assert.Nil(t, sessions[0].LastUsedAIModel)
	require.NotNil(t, sessions[1].Description)
	assert.Equal(t, "Session 1", *sessions[1].Description)
	assert.Equal(t, 1700000001, sessions[1].CreatedAt)
	assert.Equal(t, 1700001001, sessions[1].UpdatedAt)
	assert.Equal(t, 1, sessions[1].Version)
	require.NotNil(t, sessions[1].LastUsedAIModel)
	assert.Equal(t, "claude-sonnet", *sessions[1].LastUsedAIModel)
}

func TestFetchSessions_GraphQLError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))

	_, err := fetchSessions(t.Context(), c, nil, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list Infra Assistant sessions")
	assert.ErrorContains(t, err, "boom")
}

func TestSessionsTable(t *testing.T) {
	sessions := []chatSession{
		{ID: "a", UpdatedAt: 0},
		{ID: "b", Description: new("Fix VPC"), UpdatedAt: 1700000000, LastUsedAIModel: new("claude-sonnet")},
	}

	table := sessionsTable(sessions)

	require.Len(t, table, 3)
	assert.Equal(t, []string{"ID", "Title", "Updated At", "Model"}, table[0])
	assert.Equal(t, "a", table[1][0])
	assert.Empty(t, table[1][1])
	assert.Empty(t, table[1][3])
	assert.Equal(t, []string{"b", "Fix VPC"}, table[2][:2])
	assert.NotEmpty(t, table[2][2])
	assert.Equal(t, "claude-sonnet", table[2][3])
}

// fakeConversationAPI serves intentChatConversation for a single known session
// and records every request.
type fakeConversationAPI struct {
	t          *testing.T
	id         string
	transcript string
	requests   []graphqlRequest
}

func (f *fakeConversationAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assert.Equal(f.t, "/graphql", r.URL.Path)
	assert.Equal(f.t, "Bearer "+testToken, r.Header.Get("Authorization"))

	var req graphqlRequest
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&req))
	f.requests = append(f.requests, req)

	var conversation any
	if req.Variables["id"] == f.id {
		conversation = map[string]any{"transcript": f.transcript}
	}

	w.Header().Set("Content-Type", "application/json")
	require.NoError(f.t, json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"intentChatConversation": conversation},
	}))
}

func TestWriteTranscript(t *testing.T) {
	const transcript = "# Fix VPC\n\n- Messages: 24 (showing 6-15)\n\n## User\n\nhello\n"

	tests := []struct {
		name       string
		r          transcriptRange
		wantOffset float64
		wantLimit  float64
	}{
		{name: "defaults", wantOffset: 0, wantLimit: 0},
		{name: "offset and limit", r: transcriptRange{offset: 5, limit: 10}, wantOffset: 5, wantLimit: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeConversationAPI{t: t, id: "01JABC", transcript: transcript}
			c := newTestClient(t, api)

			var out bytes.Buffer
			require.NoError(t, writeTranscript(t.Context(), c, "01JABC", tt.r, &out))
			assert.Equal(t, transcript, out.String())

			require.Len(t, api.requests, 1)
			req := api.requests[0]
			assert.Contains(t, req.Query, "intentChatConversation(id: $id)")
			assert.Contains(t, req.Query, "transcript(offset: $offset, limit: $limit)")
			assert.Equal(t, "01JABC", req.Variables["id"])
			assert.InDelta(t, tt.wantOffset, req.Variables["offset"], 0)
			assert.InDelta(t, tt.wantLimit, req.Variables["limit"], 0)
		})
	}
}

func TestWriteTranscript_NotFound(t *testing.T) {
	api := &fakeConversationAPI{t: t, id: "01JABC", transcript: "unused"}
	c := newTestClient(t, api)

	var out bytes.Buffer
	err := writeTranscript(t.Context(), c, "01JOTHER", transcriptRange{}, &out)

	require.Error(t, err)
	assert.EqualError(t, err, `session "01JOTHER" not found (sessions belong to the user who created them; API key profiles can't read them)`)
	assert.Empty(t, out.String())
}

func TestWriteTranscript_GraphQLError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"offset and limit must not be negative"}],"data":null}`))
	}))

	var out bytes.Buffer
	err := writeTranscript(t.Context(), c, "01JABC", transcriptRange{}, &out)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get Infra Assistant session")
	assert.ErrorContains(t, err, "offset and limit must not be negative")
	assert.Empty(t, out.String())
}

func TestGetSession_ValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "missing id", args: []string{}, wantErr: "exactly one argument is required"},
		{name: "too many args", args: []string{"a", "b"}, wantErr: "exactly one argument is required"},
		{name: "blank id", args: []string{"  "}, wantErr: "session ID must be non-empty"},
		{name: "negative offset", args: []string{"--offset", "-1", "01JABC"}, wantErr: "offset must be non-negative"},
		{name: "negative limit", args: []string{"--limit", "-3", "01JABC"}, wantErr: "limit must be non-negative"},
		{name: "offset too large", args: []string{"--offset", "2147483648", "01JABC"}, wantErr: "offset must be at most 2147483647"},
		{name: "limit too large", args: []string{"--limit", "2147483648", "01JABC"}, wantErr: "limit must be at most 2147483647"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := &cli.Command{
				Name:   "get",
				Flags:  []cli.Flag{flagTranscriptOffset, flagTranscriptLimit},
				Action: getSession,
			}

			err := command.Run(t.Context(), append([]string{"get"}, tt.args...))
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
