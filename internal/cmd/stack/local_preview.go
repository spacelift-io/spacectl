package stack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func localPreview() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		stackID, err := getStackID(cliCtx)
		if err != nil {
			return err
		}
		ctx := context.Background()

		packagePath := ""
		if cliCtx.Bool(flagProjectRootOnly.Name) {
			packagePath, err = getGitRepositorySubdir()
			if err != nil {
				return fmt.Errorf("couldn't get the packagePath: %w", err)
			}
		}

		// TODO(lexton): this may be broken because of the side effects from the getStackID function - where it automatically moved you to the project root
		// we need to test out this functionality to ensure that it works as expected
		// Specifically I am concerned about where the matchers are loaded from - it required loaded from the project_root only
		// it won't load the .gitinore and .terrraformignore files from the current directory
		if !cliCtx.Bool(flagNoFindRepositoryRoot.Name) {
			if err := internal.MoveToRepositoryRoot(); err != nil {
				return fmt.Errorf("couldn't move to repository root: %w", err)
			}
		}

		fmt.Printf("Packing local workspace... path='%s'\n", packagePath)

		var uploadMutation struct {
			UploadLocalWorkspace struct {
				ID        string `graphql:"id"`
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

		matchFn, err := internal.GetIgnoreMatcherFn(ctx, packagePath)
		if err != nil {
			return fmt.Errorf("couldn't analyze .gitignore and .terraformignore files")
		}

		tgz := *archiver.DefaultTarGz
		tgz.ForceArchiveImplicitTopLevelFolder = true
		tgz.MatchFn = matchFn

		if err := tgz.Archive([]string{"."}, fp); err != nil {
			return fmt.Errorf("couldn't archive local directory: %w", err)
		}

		if cliCtx.Bool(flagNoUpload.Name) {
			fmt.Println("No upload flag was provided, will not create run, saved archive at:", fp)
			return nil
		}

		defer os.Remove(fp)

		fmt.Println("Uploading local workspace...")

		if err := internal.UploadArchive(ctx, uploadMutation.UploadLocalWorkspace.UploadURL, fp); err != nil {
			return fmt.Errorf("couldn't upload archive: %w", err)
		}

		var triggerMutation struct {
			RunProposeLocalWorkspace struct {
				ID string `graphql:"id"`
			} `graphql:"runProposeLocalWorkspace(stack: $stack, workspace: $workspace)"`
		}

		triggerVariables := map[string]interface{}{
			"stack":     graphql.ID(stackID),
			"workspace": graphql.ID(uploadMutation.UploadLocalWorkspace.ID),
		}

		var requestOpts []graphql.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client.Mutate(ctx, &triggerMutation, triggerVariables, requestOpts...); err != nil {
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
