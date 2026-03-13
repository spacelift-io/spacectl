package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/provider/internal"
)

var flagDocsArchive = &cli.StringFlag{
	Name:     "docs-archive",
	Usage:    "[Required] Path to a gzipped tar archive containing the documentation",
	Required: true,
}

func uploadDocs() cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		versionID := cliCmd.String(flagRequiredVersionID.Name)
		archivePath := cliCmd.String(flagDocsArchive.Name)

		data, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("could not read docs archive: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(data)

		var mutation struct {
			UploadDocs internal.Version `graphql:"terraformProviderVersionUploadDocs(version: $version, archive: $archive)"`
		}

		variables := map[string]any{
			"version": graphql.ID(versionID),
			"archive": graphql.String(encoded),
		}

		if err := authenticated.Client().Mutate(ctx, &mutation, variables); err != nil {
			return fmt.Errorf("could not upload docs for provider version: %w", err)
		}

		fmt.Printf("Documentation uploaded for provider version %s\n", mutation.UploadDocs.Number)
		return nil
	}
}
