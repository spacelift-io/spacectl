package module

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hasura/go-graphql-client"
	"github.com/mholt/archiver/v3"
	"github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/internal"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func localPreview() cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		moduleID := cliCtx.String(flagModuleID.Name)
		ctx := context.Background()

		if !cliCtx.Bool(flagNoFindRepositoryRoot.Name) {
			if err := internal.MoveToRepositoryRoot(); err != nil {
				return fmt.Errorf("couldn't move to repository root: %w", err)
			}
		}

		fmt.Println("Packing local workspace...")

		var uploadMutation struct {
			UploadLocalWorkspace struct {
				ID        string `graphql:"id"`
				UploadURL string `graphql:"uploadUrl"`
			} `graphql:"uploadLocalWorkspace(stack: $stack)"`
		}

		uploadVariables := map[string]interface{}{
			"stack": graphql.ID(moduleID),
		}

		if err := authenticated.Client.Mutate(ctx, &uploadMutation, uploadVariables); err != nil {
			return err
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
			VersionProposeLocalWorkspace []runQuery `graphql:"versionProposeLocalWorkspace(module: $module, workspace: $workspace, testIds: $testIds)"`
		}

		tests := []string{}
		for _, test := range cliCtx.StringSlice(flagTests.Name) {
			tests = append(tests, test)
		}
		triggerVariables := map[string]interface{}{
			"module":    graphql.ID(moduleID),
			"workspace": graphql.ID(uploadMutation.UploadLocalWorkspace.ID),
			"testIds":   tests,
		}

		var requestOpts []client.RequestOption
		if cliCtx.IsSet(flagRunMetadata.Name) {
			requestOpts = append(requestOpts, client.WithHeader(internal.UserProvidedRunMetadataHeader, cliCtx.String(flagRunMetadata.Name)))
		}

		if err := authenticated.Client.Mutate(ctx, &triggerMutation, triggerVariables, requestOpts...); err != nil {
			return err
		}

		model := newModuleLocalPreviewModel(moduleID, triggerMutation.VersionProposeLocalWorkspace)

		go func() {
			// Refresh run state every 5 seconds.
			ticker := time.NewTicker(time.Second * 5)
			defer ticker.Stop()
			for range ticker.C {
				newRuns := make([]runQuery, len(triggerMutation.VersionProposeLocalWorkspace))
				var g errgroup.Group
				for i := range newRuns {
					index := i
					g.Go(func() error {
						var getRun struct {
							Module struct {
								Run runQuery `graphql:"run(id: $run)"`
							} `graphql:"module(id: $module)"`
						}

						if err := authenticated.Client.Query(ctx, &getRun, map[string]interface{}{
							"module": graphql.ID(moduleID),
							"run":    graphql.ID(triggerMutation.VersionProposeLocalWorkspace[index].ID),
						}); err != nil {
							return err
						}

						newRuns[index] = getRun.Module.Run

						return nil
					})
				}
				if err := g.Wait(); err != nil {
					log.Fatal("couldn't get runs: ", err)
				}
				model.setRuns(newRuns)
			}
		}()

		// Run the UI.
		if _, err := tea.NewProgram(model).Run(); err != nil {
			fmt.Println("could not run program:", err)
			os.Exit(1)
		}

		return nil
	}
}

type checkRunsFinishedMsg struct{}

type moduleLocalPreviewModel struct {
	sync.Mutex
	ModuleID     string
	Runs         []runQuery
	SpinnersSlow []spinner.Model
	SpinnersFast []spinner.Model
}

type runQuery struct {
	ID       string `graphql:"id"`
	Title    string `graphql:"title"`
	State    string `graphql:"state"`
	Finished bool   `graphql:"finished"`
}

func newModuleLocalPreviewModel(moduleID string, runsState []runQuery) *moduleLocalPreviewModel {
	slowMiniDot := spinner.Spinner{
		Frames: []string{"⠂", "⠐"},
		FPS:    time.Second,
	}
	spinnersSlow := make([]spinner.Model, len(runsState))
	spinnersFast := make([]spinner.Model, len(runsState))
	for i := range runsState {
		spinnersSlow[i] = spinner.New(spinner.WithSpinner(slowMiniDot))
		spinnersFast[i] = spinner.New(spinner.WithSpinner(spinner.MiniDot))
	}
	return &moduleLocalPreviewModel{
		ModuleID:     moduleID,
		Runs:         runsState,
		SpinnersSlow: spinnersSlow,
		SpinnersFast: spinnersFast,
	}
}

func (m *moduleLocalPreviewModel) setRuns(newRuns []runQuery) {
	m.Lock()
	m.Runs = newRuns
	m.Unlock()
}

func (m *moduleLocalPreviewModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.SpinnersSlow {
		cmds = append(cmds, m.SpinnersSlow[i].Tick)
	}
	for i := range m.SpinnersFast {
		cmds = append(cmds, m.SpinnersFast[i].Tick)
	}
	cmds = append(cmds, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
		return checkRunsFinishedMsg{}
	}))
	return tea.Batch(cmds...)
}

func (m *moduleLocalPreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Lock()
	defer m.Unlock()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		for i := range m.SpinnersSlow {
			if m.SpinnersSlow[i].ID() == msg.ID {
				m.SpinnersSlow[i], cmd = m.SpinnersSlow[i].Update(msg)
				break
			}
		}
		for i := range m.SpinnersFast {
			if m.SpinnersFast[i].ID() == msg.ID {
				m.SpinnersFast[i], cmd = m.SpinnersFast[i].Update(msg)
				break
			}
		}
		return m, cmd
	case checkRunsFinishedMsg:
		allTerminated := true
		for _, run := range m.Runs {
			if !run.Finished {
				allTerminated = false
				break
			}
		}
		if allTerminated {
			return m, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
				fmt.Println()
				return tea.Quit()
			})
		}

		return m, tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
			return checkRunsFinishedMsg{}
		})
	default:
		return m, nil
	}
}

func (m *moduleLocalPreviewModel) View() (s string) {
	m.Lock()
	defer m.Unlock()

	s += "\n"
	for i := range m.Runs {
		spinnerView := m.SpinnersSlow[i].View()
		if m.Runs[i].State != "QUEUED" {
			spinnerView = m.SpinnersFast[i].View()
		}
		if m.Runs[i].Finished {
			spinnerView = "⠿"
		}
		s += fmt.Sprintf(" %s %s • %s • %s\n", spinnerView, styledState(m.Runs[i].State), m.Runs[i].Title, authenticated.Client.URL(
			"/module/%s/run/%s",
			m.ModuleID,
			m.Runs[i].ID,
		))
	}
	s += "\n"
	return
}

func styledState(state string) string {
	var (
		inProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render
		finishedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("70")).Render
		failedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render
	)

	switch state {
	case "QUEUED":
		return state
	case "FINISHED":
		return finishedStyle(state)
	case "FAILED":
		return failedStyle(state)
	default:
		return inProgressStyle(state)
	}
}
