package stack

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/logs"
	"github.com/spacelift-io/spacectl/internal/nullable"
)

func getStackForLocalPreview(ctx context.Context, cliCmd *cli.Command) (*stack, error) {
	if !cliCmd.Bool(flagOnlyEnabled.Name) {
		s, err := getStack[stack](ctx, cliCmd)
		if err != nil {
			return nil, err
		}
		if !s.LocalPreviewEnabled {
			linkToStack := authenticated.Client().URL("/stack/%s", s.ID)
			return nil, fmt.Errorf("local preview has not been enabled for this stack, please enable local preview in the stack settings: %s", linkToStack)
		}

		return s, nil
	}

	s, err := getStackFiltered(ctx, cliCmd, func(s *stack) bool {
		return s.LocalPreviewEnabled
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func localPreview(useHeaders bool) cli.ActionFunc {
	return func(ctx context.Context, cliCmd *cli.Command) error {
		envVars, err := parseEnvVariablesForLocalPreview(cliCmd)
		if err != nil {
			return err
		}

		s, err := getStackForLocalPreview(ctx, cliCmd)
		if err != nil {
			return err
		}

		var packagePath *string
		if cliCmd.Bool(flagProjectRootOnly.Name) {
			root, err := getGitRepositorySubdir()
			if err != nil {
				return fmt.Errorf("couldn't get the packagePath: %w", err)
			}
			packagePath = &root
		}

		var runMetadata *string
		if cliCmd.IsSet(flagRunMetadata.Name) {
			runMetadata = nullable.OfValue(cliCmd.String(flagRunMetadata.Name))
		}

		runID, err := createLocalPreviewRun(
			ctx,
			LocalPreviewOptions{
				StackID:            s.ID,
				EnvironmentVars:    envVars,
				Targets:            cliCmd.StringSlice(flagTarget.Name),
				Path:               packagePath,
				FindRepositoryRoot: !cliCmd.Bool(flagNoFindRepositoryRoot.Name),
				DisregardGitignore: cliCmd.IsSet(flagDisregardGitignore.Name),
				UseHeaders:         useHeaders,
				NoUpload:           cliCmd.Bool(flagNoUpload.Name),
				RunMetadata:        runMetadata,
				PrioritizeRun:      cliCmd.Bool(flagPrioritizeRun.Name),
				ShowUploadProgress: true,
				IncludeGitDir:      cliCmd.Bool(flagWithGitDir.Name),
			},
			os.Stdout,
		)

		if err != nil {
			return fmt.Errorf("failed to create local preview run: %w", err)
		}

		linkToRun := authenticated.Client().URL(
			"/stack/%s/run/%s",
			s.ID,
			runID,
		)
		fmt.Println("The live run can be visited at", linkToRun)

		if cliCmd.Bool(flagNoTail.Name) {
			return nil
		}

		terminal, err := logs.NewExplorer(s.ID, runID).RunFilteredLogs(ctx)
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

func parseEnvVariablesForLocalPreview(cliCmd *cli.Command) ([]EnvironmentVariable, error) {
	envVars := make([]EnvironmentVariable, 0)

	var err error
	for _, ev := range cliCmd.StringSlice(flagOverrideEnvVars.Name) {
		if envVars, err = parseEnvVar(ev, envVars, nil); err != nil {
			return nil, err
		}
	}

	for _, ev := range cliCmd.StringSlice(flagOverrideEnvVarsTF.Name) {
		if envVars, err = parseEnvVar(ev, envVars, func(s string) string {
			return strings.Join([]string{"TF_", s}, "")
		}); err != nil {
			return nil, err
		}
	}

	return envVars, nil
}

type LocalPreviewOptions struct {
	StackID            string
	EnvironmentVars    []EnvironmentVariable
	Targets            []string
	Path               *string
	FindRepositoryRoot bool
	DisregardGitignore bool
	UseHeaders         bool
	NoUpload           bool
	RunMetadata        *string
	PrioritizeRun      bool
	ShowUploadProgress bool
	IncludeGitDir      bool
}

func createLocalPreviewRun(
	ctx context.Context,
	options LocalPreviewOptions,
	writer io.Writer,
) (string, error) {
	envVars := options.EnvironmentVars

	var val string
	for _, v := range options.Targets {
		val = strings.Join([]string{val, "-target=" + v}, " ")
	}

	if len(options.Targets) > 0 {
		envVars = append(envVars, EnvironmentVariable{
			Key:   "TF_CLI_ARGS_plan",
			Value: graphql.String(strings.TrimSpace(val)),
		})
	}

	packagePath := options.Path
	if options.FindRepositoryRoot {
		if err := internal.MoveToRepositoryRoot(); err != nil {
			return "", fmt.Errorf("couldn't move to repository root: %w", err)
		}
	}

	if packagePath == nil {
		fmt.Fprintln(writer, "Packing local workspace...")
	} else {
		fmt.Fprintf(writer, "Packing '%s' as local workspace...\n", *packagePath)
	}

	// Define concrete types
	type basicResponse struct {
		ID        string `graphql:"id"`
		UploadURL string `graphql:"uploadUrl"`
	}

	type headersResponse struct {
		ID            string            `graphql:"id"`
		UploadURL     string            `graphql:"uploadUrl"`
		UploadHeaders structs.StringMap `graphql:"uploadHeaders"`
	}

	var workspaceID string
	var uploadURL string
	var headers map[string]string

	uploadVariables := map[string]any{
		"stack": graphql.ID(options.StackID),
	}

	if options.UseHeaders {
		// Use the headers response struct
		var headersMutation struct {
			UploadLocalWorkspace headersResponse `graphql:"uploadLocalWorkspace(stack: $stack)"`
		}

		if err := authenticated.Client().Mutate(ctx, &headersMutation, uploadVariables); err != nil {
			return "", fmt.Errorf("failed to upload local workspace: %w", err)
		}

		workspaceID = headersMutation.UploadLocalWorkspace.ID
		uploadURL = headersMutation.UploadLocalWorkspace.UploadURL
		headers = headersMutation.UploadLocalWorkspace.UploadHeaders.StdMap()
	} else {
		// Use the basic response struct
		var basicMutation struct {
			UploadLocalWorkspace basicResponse `graphql:"uploadLocalWorkspace(stack: $stack)"`
		}

		if err := authenticated.Client().Mutate(ctx, &basicMutation, uploadVariables); err != nil {
			return "", fmt.Errorf("failed to upload local workspace: %w", err)
		}

		workspaceID = basicMutation.UploadLocalWorkspace.ID
		uploadURL = basicMutation.UploadLocalWorkspace.UploadURL
		headers = nil
	}

	fp := filepath.Join(os.TempDir(), "spacectl", "local-workspace", fmt.Sprintf("%s.tar.gz", workspaceID))

	ignoreFiles := []string{".terraformignore"}
	if !options.DisregardGitignore {
		ignoreFiles = append(ignoreFiles, ".gitignore")
	}

	matchFn, err := internal.GetIgnoreMatcherFn(ctx, packagePath, ignoreFiles, options.IncludeGitDir)
	if err != nil {
		return "", fmt.Errorf("couldn't analyze .gitignore and .terraformignore files")
	}

	tgz := *archiver.DefaultTarGz
	tgz.ForceArchiveImplicitTopLevelFolder = true
	tgz.MatchFn = matchFn

	if err := tgz.Archive([]string{"."}, fp); err != nil {
		return "", fmt.Errorf("couldn't archive local directory: %w", err)
	}

	if options.NoUpload {
		fmt.Fprintf(writer, "No upload flag was provided, will not create run, saved archive at: %s\n", fp)
		return "", nil
	}

	defer os.Remove(fp)

	fmt.Fprintln(writer, "Uploading local workspace...")

	if err := internal.UploadArchive(ctx, uploadURL, fp, headers, options.ShowUploadProgress); err != nil {
		return "", fmt.Errorf("couldn't upload archive: %w", err)
	}

	var triggerMutation struct {
		RunProposeLocalWorkspace struct {
			ID string `graphql:"id"`
		} `graphql:"runProposeLocalWorkspace(stack: $stack, workspace: $workspace, environmentVarsOverrides: $environmentVarsOverrides)"`
	}

	triggerVariables := map[string]any{
		"stack":                    graphql.ID(options.StackID),
		"workspace":                graphql.ID(workspaceID),
		"environmentVarsOverrides": envVars,
	}

	var requestOpts []graphql.RequestOption
	if options.RunMetadata != nil {
		requestOpts = append(requestOpts, graphql.WithHeader(internal.UserProvidedRunMetadataHeader, *options.RunMetadata))
	}

	fmt.Fprintln(writer, "Creating local preview run...")
	if err = authenticated.Client().Mutate(ctx, &triggerMutation, triggerVariables, requestOpts...); err != nil {
		return "", err
	}

	fmt.Fprintln(writer, "You have successfully created a local preview run!")

	if options.PrioritizeRun {
		_, err = setRunPriority(ctx, options.StackID, triggerMutation.RunProposeLocalWorkspace.ID, true)
		if err != nil {
			fmt.Fprintln(writer, "Failed to prioritize the run due to err:", err)
			fmt.Fprintln(writer, "Resolve the issue and prioritize the run manually")
		} else {
			fmt.Fprintln(writer, "The run has been successfully prioritized!")
		}
	}

	return triggerMutation.RunProposeLocalWorkspace.ID, nil
}
