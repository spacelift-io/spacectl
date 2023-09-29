package stack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

func localPreview() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		envVars, err := parseEnvVariablesForLocalPreview(cliCtx)
		if err != nil {
			return err
		}

		stackID, err := getStackID(cliCtx)
		if err != nil {
			return err
		}

		ctx := context.Background()

		var packagePath *string = nil
		if cliCtx.Bool(flagProjectRootOnly.Name) {
			root, err := getGitRepositorySubdir()
			if err != nil {
				return fmt.Errorf("couldn't get the packagePath: %w", err)
			}
			packagePath = &root
		}
		if !cliCtx.Bool(flagNoFindRepositoryRoot.Name) {
			if err := internal.MoveToRepositoryRoot(); err != nil {
				return fmt.Errorf("couldn't move to repository root: %w", err)
			}
		}

		if packagePath == nil {
			fmt.Printf("Packing local workspace...\n")
		} else {
			fmt.Printf("Packing '%s' as local workspace...\n", *packagePath)
		}

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
			return fmt.Errorf("failed to upload local workspace: %w", err)
		}

		fp := filepath.Join(os.TempDir(), "spacectl", "local-workspace", fmt.Sprintf("%s.tar.gz", uploadMutation.UploadLocalWorkspace.ID))

		ignoreFiles := []string{".terraformignore"}
		if !cliCtx.IsSet(flagDisregardGitignore.Name) {
			ignoreFiles = append(ignoreFiles, ".gitignore")
		}

		matchFn, err := internal.GetIgnoreMatcherFn(ctx, nil, ignoreFiles)
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
			} `graphql:"runProposeLocalWorkspace(stack: $stack, workspace: $workspace, environmentVarsOverrides: $environmentVarsOverrides)"`
		}

		triggerVariables := map[string]interface{}{
			"stack":                    graphql.ID(stackID),
			"workspace":                graphql.ID(uploadMutation.UploadLocalWorkspace.ID),
			"environmentVarsOverrides": envVars,
		}

		var requestOpts []graphql.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client.Mutate(ctx, &triggerMutation, triggerVariables, requestOpts...); err != nil {
			return err
		}

		fmt.Println("You have successfully created a local preview run!")

		linkToRun := authenticated.Client.URL(
			"/stack/%s/run/%s",
			stackID,
			triggerMutation.RunProposeLocalWorkspace.ID,
		)
		fmt.Println("The live run can be visited at", linkToRun)

		if cliCtx.Bool(flagNoTail.Name) {
			return nil
		}

		terminal, err := runLogsWithAction(ctx, stackID, triggerMutation.RunProposeLocalWorkspace.ID, nil)
		if err != nil {
			return err
		}

		fmt.Println("View full logs at", linkToRun)

		return terminal.Error()
	}
}

// EnvironmentVariable represents a key-value pair of environment variables
type EnvironmentVariable struct {
	Key   graphql.String `json:"key"`
	Value graphql.String `json:"value"`
}

func parseEnvVariablesForLocalPreview(cliCtx *cli.Context) ([]EnvironmentVariable, error) {
	envVars := make([]EnvironmentVariable, 0)

	parseFn := func(ev string, mutateKey func(string) string) error {
		parts := strings.Split(ev, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable %q, must be in the form of KEY=VALUE", ev)
		}

		if mutateKey != nil {
			parts[0] = mutateKey(parts[0])
		}

		envVars = append(envVars, EnvironmentVariable{
			Key:   graphql.String(parts[0]),
			Value: graphql.String(parts[1]),
		})

		return nil
	}

	for _, ev := range cliCtx.StringSlice(flagOverrideEnvVars.Name) {
		if err := parseFn(ev, nil); err != nil {
			return nil, err
		}
	}

	for _, ev := range cliCtx.StringSlice(flagOverrideEnvVarsTF.Name) {
		if err := parseFn(ev, func(s string) string {
			return strings.Join([]string{"TF_", s}, "")
		}); err != nil {
			return nil, err
		}
	}

	return envVars, nil
}
