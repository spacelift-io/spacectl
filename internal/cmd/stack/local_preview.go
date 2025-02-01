package stack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
)

func localPreview() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		envVars, err := parseEnvVariablesForLocalPreview(cliCtx)
		if err != nil {
			return err
		}

		if got := cliCtx.StringSlice(flagTarget.Name); len(got) > 0 {
			var val string
			for _, v := range got {
				val = strings.Join([]string{val, "-target=" + v}, " ")
			}

			envVars = append(envVars, EnvironmentVariable{
				Key:   "TF_CLI_ARGS_plan",
				Value: graphql.String(strings.TrimSpace(val)),
			})
		}

		stack, err := getStack[stack](cliCtx)
		if err != nil {
			return err
		}

		if !stack.LocalPreviewEnabled {
			linkToStack := authenticated.Client.URL("/stack/%s", stack.ID)
			return fmt.Errorf("local preview has not been enabled for this stack, please enable local preview in the stack settings: %s", linkToStack)
		}

		ctx := context.Background()

		var packagePath *string
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
			"stack": graphql.ID(stack.ID),
		}

		if err := authenticated.Client.Mutate(ctx, &uploadMutation, uploadVariables); err != nil {
			return fmt.Errorf("failed to upload local workspace: %w", err)
		}

		fp := filepath.Join(os.TempDir(), "spacectl", "local-workspace", fmt.Sprintf("%s.tar.gz", uploadMutation.UploadLocalWorkspace.ID))

		ignoreFiles := []string{".terraformignore"}
		if !cliCtx.IsSet(flagDisregardGitignore.Name) {
			ignoreFiles = append(ignoreFiles, ".gitignore")
		}

		matchFn, err := internal.GetIgnoreMatcherFn(ctx, packagePath, ignoreFiles)
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
			"stack":                    graphql.ID(stack.ID),
			"workspace":                graphql.ID(uploadMutation.UploadLocalWorkspace.ID),
			"environmentVarsOverrides": envVars,
		}

		var requestOpts []graphql.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err = authenticated.Client.Mutate(ctx, &triggerMutation, triggerVariables, requestOpts...); err != nil {
			return err
		}

		fmt.Println("You have successfully created a local preview run!")

		if cliCtx.Bool(flagPrioritizeRun.Name) {
			_, err = setRunPriority(cliCtx, stack.ID, triggerMutation.RunProposeLocalWorkspace.ID, true)
			if err != nil {
				fmt.Printf("Failed to prioritize the run due to err: %v\n", err)
				fmt.Println("Resolve the issue and prioritize the run manually")
			} else {
				fmt.Println("The run has been successfully prioritized!")
			}
		}

		linkToRun := authenticated.Client.URL(
			"/stack/%s/run/%s",
			stack.ID,
			triggerMutation.RunProposeLocalWorkspace.ID,
		)
		fmt.Println("The live run can be visited at", linkToRun)

		if cliCtx.Bool(flagNoTail.Name) {
			return nil
		}

		terminal, err := runLogsWithAction(ctx, stack.ID, triggerMutation.RunProposeLocalWorkspace.ID, nil)
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

func parseEnvVar(env string, envVars []EnvironmentVariable, mutateKey func(string) string) ([]EnvironmentVariable, error) {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) != 2 {
		return envVars, fmt.Errorf("invalid environment variable %q, must be in the form of KEY=VALUE", env)
	}

	if mutateKey != nil {
		parts[0] = mutateKey(parts[0])
	}

	return append(envVars, EnvironmentVariable{
		Key:   graphql.String(parts[0]),
		Value: graphql.String(parts[1]),
	}), nil
}

func parseEnvVariablesForLocalPreview(cliCtx *cli.Context) ([]EnvironmentVariable, error) {
	envVars := make([]EnvironmentVariable, 0)

	var err error
	for _, ev := range cliCtx.StringSlice(flagOverrideEnvVars.Name) {
		if envVars, err = parseEnvVar(ev, envVars, nil); err != nil {
			return nil, err
		}
	}

	for _, ev := range cliCtx.StringSlice(flagOverrideEnvVarsTF.Name) {
		if envVars, err = parseEnvVar(ev, envVars, func(s string) string {
			return strings.Join([]string{"TF_", s}, "")
		}); err != nil {
			return nil, err
		}
	}

	return envVars, nil
}
