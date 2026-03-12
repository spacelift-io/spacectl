package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type apiRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

type gqlErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func run(ctx context.Context, cliCmd *cli.Command) error {
	schemaOnly := cliCmd.Bool(flagSchema.Name)
	query, variables, operation, schemaSelector, err := resolveRequestParts(cliCmd, schemaOnly)
	if err != nil {
		return err
	}

	reqBody := apiRequest{
		Query:         query,
		Variables:     variables,
		OperationName: operation,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/graphql", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient, token, endpoint, err := authenticated.HTTPClient(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	baseURL, err := url.Parse(strings.TrimRight(endpoint, "/graphql"))
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}
	req.URL.Scheme = baseURL.Scheme
	req.URL.Host = baseURL.Host

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if schemaOnly {
		if err := outputSchema(body, schemaSelector); err != nil {
			return err
		}
	} else {
		if err := outputResponse(body, cliCmd.Bool(flagRaw.Name)); err != nil {
			return err
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthorized: you can re-login using `spacectl profile login`")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, snippet(body))
	}

	if msg, ok := graphqlErrorMessage(body); ok {
		return fmt.Errorf("graphql errors: %s", msg)
	}

	return nil
}

func resolveRequestParts(cliCmd *cli.Command, schemaOnly bool) (string, map[string]any, string, string, error) {
	query := strings.TrimSpace(cliCmd.String(flagQuery.Name))
	file := strings.TrimSpace(cliCmd.String(flagFile.Name))
	variablesRaw := cliCmd.String(flagVariables.Name)
	operation := strings.TrimSpace(cliCmd.String(flagOperation.Name))
	argsQuery := strings.TrimSpace(strings.Join(cliCmd.Args().Slice(), " "))

	if err := validateSchemaArgs(schemaOnly, query, file, variablesRaw, operation, argsQuery); err != nil {
		return "", nil, "", "", err
	}

	if schemaOnly {
		return introspectionQuery, nil, "", argsQuery, nil
	}

	if query == "" && file == "" && argsQuery != "" {
		return normalizeQuery(argsQuery), nil, operation, "", nil
	}

	resolvedQuery, err := resolveQueryFrom(query, file, os.Stdin, isatty.IsTerminal(os.Stdin.Fd()))
	if err != nil {
		return "", nil, "", "", err
	}

	variables, err := parseVariables(variablesRaw)
	if err != nil {
		return "", nil, "", "", err
	}

	return resolvedQuery, variables, operation, "", nil
}

func resolveQueryFrom(query string, file string, stdin io.Reader, stdinIsTerminal bool) (string, error) {
	if query != "" && file != "" {
		return "", errors.New("only one of --query or --file can be specified")
	}

	if query == "-" {
		return readQueryFromStdin(stdin)
	}

	if query != "" {
		return query, nil
	}

	if file != "" {
		contents, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return strings.TrimSpace(string(contents)), nil
	}

	if !stdinIsTerminal {
		return readQueryFromStdin(stdin)
	}

	return "", errors.New("query required: use --query, --file, or stdin")
}

func readQueryFromStdin(stdin io.Reader) (string, error) {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	query := strings.TrimSpace(string(data))
	if query == "" {
		return "", errors.New("stdin is empty")
	}
	return query, nil
}

func parseVariables(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("failed to parse variables JSON: %w", err)
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("variables must be a JSON object")
	}

	return obj, nil
}

func validateSchemaArgs(schemaOnly bool, query string, file string, variables string, operation string, argsQuery string) error {
	if !schemaOnly {
		return nil
	}

	if query != "" || file != "" || strings.TrimSpace(variables) != "" || operation != "" {
		return errors.New("--schema cannot be combined with query, file, variables, or operation flags")
	}

	return nil
}

func outputResponse(body []byte, raw bool) error {
	if raw || !isatty.IsTerminal(os.Stdout.Fd()) {
		_, err := os.Stdout.Write(body)
		return err
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		_, werr := os.Stdout.Write(body)
		return werr
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func graphqlErrorMessage(body []byte) (string, bool) {
	var gqlErr gqlErrorResponse
	if err := json.Unmarshal(body, &gqlErr); err != nil {
		return "", false
	}
	if len(gqlErr.Errors) == 0 {
		return "", false
	}
	if gqlErr.Errors[0].Message != "" {
		return gqlErr.Errors[0].Message, true
	}
	return "unknown error", true
}

func snippet(body []byte) string {
	const limit = 512
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) <= limit {
		return string(trimmed)
	}
	return string(trimmed[:limit]) + "..."
}

func normalizeQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "query") || strings.HasPrefix(lower, "mutation") || strings.HasPrefix(lower, "subscription") || strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	return "query { " + trimmed + " }"
}
