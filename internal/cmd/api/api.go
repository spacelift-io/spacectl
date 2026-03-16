package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var (
	flagVariables = &cli.StringFlag{
		Name:  "variables",
		Usage: "JSON object with GraphQL variables",
	}
	flagRaw = &cli.BoolFlag{
		Name:  "raw",
		Usage: "Output response body without pretty-printing",
	}
)

func Command() cmd.Command {
	return cmd.Command{
		Name:     "api",
		Usage:    "Call the Spacelift GraphQL API",
		Category: "GraphQL",
		Versions: []cmd.VersionedCommand{
			{
				EarliestVersion: cmd.SupportedVersionAll,
				Command: &cli.Command{
					Before:      authenticated.Ensure,
					Flags:       []cli.Flag{flagVariables, flagRaw},
					Description: "Pass a read-only GraphQL query as a positional argument, or pipe via stdin.\nBare field selections are wrapped as query { ... } automatically.\nMutations and subscriptions are not supported. Read more: https://github.com/spacelift-io/spacectl/tree/main/internal/cmd/api/README.md",
					ArgsUsage:   "[query]",
					Action:      run,
				},
			},
		},
	}
}

type apiRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

var errMutationsNotAllowed = errors.New("mutations are not supported by spacectl api")

func run(ctx context.Context, cliCmd *cli.Command) error {
	query, err := resolveQuery(cliCmd)
	if err != nil {
		return err
	}

	// This is a hopefull check but it's enough for most cases.
	// We could in theory allow mutations, but spacectl is not a great tool for
	// that so we do not.
	if isMutation(query) {
		return errMutationsNotAllowed
	}

	variables, err := parseVariables(cliCmd.String(flagVariables.Name))
	if err != nil {
		return err
	}

	payload, err := json.Marshal(apiRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/graphql", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authenticated.Client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(body, 512))
	}

	gqlErr := graphqlErrors(body)

	if err := outputJSON(body, cliCmd.Bool(flagRaw.Name)); err != nil {
		return err
	}

	if gqlErr != "" {
		fmt.Fprintf(os.Stderr, "graphql error: %s\n", gqlErr)
		return cli.Exit("", 1)
	}

	return nil
}

func resolveQuery(cliCmd *cli.Command) (string, error) {
	if args := strings.TrimSpace(strings.Join(cliCmd.Args().Slice(), " ")); args != "" {
		return normalizeQuery(args), nil
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		if q := strings.TrimSpace(string(data)); q != "" {
			return q, nil
		}
	}

	return "", errors.New("query required: pass as argument or pipe via stdin")
}

func parseVariables(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, fmt.Errorf("failed to parse variables JSON: %w", err)
	}
	return obj, nil
}

func normalizeQuery(query string) string {
	lower := strings.ToLower(query)
	if strings.HasPrefix(lower, "query") || strings.HasPrefix(lower, "mutation") || strings.HasPrefix(lower, "subscription") || strings.HasPrefix(query, "{") {
		return query
	}
	return "query { " + query + " }"
}

func isMutation(query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	return strings.HasPrefix(lower, "mutation")
}

func graphqlErrors(body []byte) string {
	var resp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &resp) != nil || len(resp.Errors) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(resp.Errors))
	for _, e := range resp.Errors {
		if e.Message != "" {
			msgs = append(msgs, e.Message)
		}
	}
	return strings.Join(msgs, "; ")
}

func outputJSON(body []byte, raw bool) error {
	if raw || !isatty.IsTerminal(os.Stdout.Fd()) {
		_, err := os.Stdout.Write(body)
		return err
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		_, err := os.Stdout.Write(body)
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
