package profile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func usageViewCSVCommand() *cli.Command {
	return &cli.Command{
		Name:      "usage-csv",
		Usage:     "Prints CSV with usage data for the current account",
		ArgsUsage: cmd.EmptyArgsUsage,
		Flags: []cli.Flag{
			flagUsageViewCSVSince,
			flagUsageViewCSVUntil,
			flagUsageViewCSVAspect,
			flagUsageViewCSVGroupBy,
			flagUsageViewCSVFile,
		},
		Before: authenticated.Ensure,
		Action: usageViewCsv,
	}
}

func usageViewCsv(ctx context.Context, cmd *cli.Command) error {
	// prep http query
	params := buildQueryParams(cmd)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/usageanalytics/csv", nil)
	if err != nil {
		return fmt.Errorf("failed to create an HTTP request: %w", err)
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// execute http query
	fmt.Fprint(os.Stderr, "Querying Spacelift for usage data...\n")
	resp, err := authenticated.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// process a response
	var filePath string
	if cmd.IsSet(flagUsageViewCSVFile.Name) {
		filePath = cmd.String(flagUsageViewCSVFile.Name)
		fd, err := os.OpenFile(filepath.Clean(filePath), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("failed to open a file descriptor: %w", err)
		}
		defer fd.Close()
		bfd := bufio.NewWriter(fd)
		defer bfd.Flush()
		_, err = io.Copy(bfd, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write the response to a file: %w", err)
		}
	} else {
		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write the response to stdout: %w", err)
		}
	}

	fmt.Fprint(os.Stderr, "Usage data downloaded successfully.\n")
	return nil
}

func buildQueryParams(cmd *cli.Command) map[string]string {
	params := make(map[string]string)

	params["since"] = cmd.String(flagUsageViewCSVSince.Name)
	params["until"] = cmd.String(flagUsageViewCSVUntil.Name)
	params["aspect"] = cmd.String(flagUsageViewCSVAspect.Name)

	if cmd.String(flagUsageViewCSVAspect.Name) == "run-minutes" {
		params["groupBy"] = cmd.String(flagUsageViewCSVGroupBy.Name)
	}

	return params
}
