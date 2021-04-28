package stack

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
	"github.com/mholt/archiver/v3"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func localPreview() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		ctx := context.Background()

		fmt.Println("Uploading local workspace...")

		var uploadMutation struct {
			UploadLocalWorkspace struct {
				ID        string `grapqhl:"id"`
				UploadURL string `graphql:"uploadUrl"`
			} `graphql:"uploadLocalWorkspace(stack: $stack)"`
		}

		uploadVariables := map[string]interface{}{
			"stack": graphql.ID(stackID),
		}

		if err := authenticated.Client.Mutate(ctx, &uploadMutation, uploadVariables); err != nil {
			return err
		}

		fp := filepath.Join(os.TempDir(), "spacectl", "local-workspace", fmt.Sprintf("%s.tar.gz", uploadMutation.UploadLocalWorkspace.ID))

		matchFn, err := getIgnoreMatcherFn(ctx)
		if err != nil {
			return fmt.Errorf("couldn't analyze .gitignore and .terraformignore files")
		}

		tgz := *archiver.DefaultTarGz
		tgz.MatchFn = matchFn

		if err := tgz.Archive([]string{"."}, fp); err != nil {
			return fmt.Errorf("couldn't archive local directory: %w", err)
		}
		defer os.Remove(fp)

		if err := uploadArchive(ctx, uploadMutation.UploadLocalWorkspace.UploadURL, fp); err != nil {
			return fmt.Errorf("couldn't upload archive: %w", err)
		}

		var triggerMutation struct {
			RunProposeLocalWorkspace struct {
				ID string `grapqhl:"id"`
			} `graphql:"runProposeLocalWorkspace(stack: $stack, workspace: $workspace)"`
		}

		triggerVariables := map[string]interface{}{
			"stack": graphql.ID(stackID),
			"workspace": graphql.ID(uploadMutation.UploadLocalWorkspace.ID),
		}

		if err := authenticated.Client.Mutate(ctx, &triggerMutation, triggerVariables); err != nil {
			return err
		}

		fmt.Println("You have successfully created a local preview run!")

		fmt.Println("The live run can be visited at", authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			triggerMutation.RunProposeLocalWorkspace.ID,
		))

		if cliCtx.Bool(flagNoTail.Name) {
			return nil
		}

		terminal, err := runLogs(ctx, stackID, triggerMutation.RunProposeLocalWorkspace.ID)
		if err != nil {
			return err
		}

		return terminal.Error()
	}
}

func getIgnoreMatcherFn(ctx context.Context) (func(filePath string) bool, error) {
	gitignore, err := ignore.CompileIgnoreFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("couldn't compile .gitignore file: %w", err)
	}
	terraformignore, err := ignore.CompileIgnoreFile(".terraformignore")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("couldn't compile .terraformignore file: %w", err)
	}
	customignore := ignore.CompileIgnoreLines(".git", ".terraform")
	return func(filePath string) bool {
		if customignore.MatchesPath(filePath) {
			return false
		}
		if gitignore != nil && gitignore.MatchesPath(filePath) {
			return false
		}
		if terraformignore != nil && terraformignore.MatchesPath(filePath) {
			return false
		}
		return true
	}, nil
}

func uploadArchive(ctx context.Context, uploadURL, path string) (err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("couldn't stat archive file: %w", err)
	}

	// #nosec G304
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("couldn't open archive file: %w", err)
	}

	bar := pb.Full.Start64(stat.Size())
	barReader := bar.NewProxyReader(f)
	defer bar.Finish()

	req, err := http.NewRequest(http.MethodPut, uploadURL, barReader)
	if err != nil {
		return fmt.Errorf("couldn't create upload request: %w", err)
	}
	req.ContentLength = stat.Size()
	req = req.WithContext(ctx)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't upload workspace: %w", err)
	}
	defer response.Body.Close()
	if code := response.StatusCode; code != http.StatusOK {
		return fmt.Errorf("unexpected response code when uploading workspace: %d", code)
	}

	return nil
}
