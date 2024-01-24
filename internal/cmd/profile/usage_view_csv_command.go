package profile

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type queryArgs struct {
	since   string
	until   string
	aspect  string
	groupBy string
}

func usageViewCSVCommand() *cli.Command {
	return &cli.Command{
		Name:      "usage-view-csv",
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
		Action: func(ctx *cli.Context) error {
			// prep http query
			args := &queryArgs{
				since:   ctx.String(flagUsageViewCSVSince.Name),
				until:   ctx.String(flagUsageViewCSVUntil.Name),
				aspect:  ctx.String(flagUsageViewCSVAspect.Name),
				groupBy: ctx.String(flagUsageViewCSVGroupBy.Name),
			}
			params := buildQueryParams(args)
			req, err := http.NewRequestWithContext(ctx.Context, http.MethodGet, "/usageanalytics/csv", nil)
			if err != nil {
				return fmt.Errorf("failed to create an HTTP request: %w", err)
			}
			q := req.URL.Query()
			for k, v := range params {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// execute http query
			log.Println("Querying Spacelift for usage data...")
			resp, err := authenticated.Client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			// save response to a file
			var filePath string
			if !ctx.IsSet(flagUsageViewCSVFile.Name) {
				basePath, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get base path: %w", err)
				}
				filePath = fmt.Sprintf(basePath+"/usage-%s-%s-%s.csv", args.aspect, args.since, args.until)
			} else {
				filePath = ctx.String(flagUsageViewCSVFile.Name)
			}
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
			log.Println("Usage data saved to", filePath)
			return nil
		},
	}
}

func buildQueryParams(args *queryArgs) map[string]string {
	params := make(map[string]string)

	params["since"] = args.since
	params["until"] = args.until
	params["aspect"] = args.aspect

	if args.aspect == "run-minutes" {
		params["groupBy"] = args.groupBy
	}

	return params
}
