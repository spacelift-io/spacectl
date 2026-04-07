package stack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

var flagOutputFile = &cli.StringFlag{
	Name:    "output",
	Aliases: []string{"o"},
	Usage:   "[Optional] `PATH` to write the state file to (defaults to stdout)",
}

func statePull() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		stackID, err := getStackID(ctx, cliCmd)
		if err != nil {
			return err
		}

		var mutation struct {
			StateDownloadURL struct {
				URL string `graphql:"url"`
			} `graphql:"stateDownloadUrl(input: $input)"`
		}

		variables := map[string]any{
			"input": StateDownloadUrlInput{StackID: graphql.ID(stackID)},
		}

		if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
			return fmt.Errorf("failed to get state download URL for stack %q: %w", stackID, err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, mutation.StateDownloadURL.URL, nil)
		if err != nil {
			return fmt.Errorf("failed to create download request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download state: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return fmt.Errorf("failed to download state: HTTP %d: %s", resp.StatusCode, string(body))
		}

		outputPath := cliCmd.String(flagOutputFile.Name)
		if outputPath == "" {
			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				return fmt.Errorf("failed to write state: %w", err)
			}
			return nil
		}

		f, err := os.OpenFile(filepath.Clean(outputPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			os.Remove(filepath.Clean(outputPath))
			return fmt.Errorf("failed to write state: %w", err)
		}

		if err := f.Close(); err != nil {
			os.Remove(filepath.Clean(outputPath))
			return fmt.Errorf("failed to flush state file: %w", err)
		}

		return nil
	}
}

type StateDownloadUrlInput struct { //nolint:staticcheck // type name must match GraphQL schema exactly
	StackID graphql.ID `json:"stackId"`
}
