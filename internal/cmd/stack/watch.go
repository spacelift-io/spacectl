package stack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/table"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/spacelift-io/spacectl/client/structs"
	"github.com/spacelift-io/spacectl/internal/cmd"
	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
	"github.com/spacelift-io/spacectl/internal/cmd/draw"
	"github.com/urfave/cli/v2"
)

func watch(cliCtx *cli.Context) error {
	input := structs.SearchInput{
		OrderBy: &structs.QueryOrder{
			Field:     "starred",
			Direction: "DESC",
		},
	}

	st := &Stacks{input}
	t, err := draw.NewTable(cliCtx.Context, st)
	if err != nil {
		return err
	}

	return t.DrawTable()
}

type Stacks struct {
	si structs.SearchInput
}

// Selected opens the selected worker pool in the browser.
func (q *Stacks) Filtered(s string) error {
	fullTextSearch := graphql.NewString(graphql.String(s))
	q.si.FullTextSearch = fullTextSearch

	return nil
}

// Selected opens the selected worker pool in the browser.
func (q *Stacks) Selected(row table.Row) ([]table.Row, error) {
	switch row[0] {
	case "Inspect":
		ctx := context.Background()
		st := &StackWatch{row[1]}
		t, err := draw.NewTable(ctx, st)
		if err != nil {
			return nil, err
		}

		return nil, t.DrawTable()
	case "Trigger":
		var mutation struct {
			RunTrigger struct {
				ID string `graphql:"id"`
			} `graphql:"runTrigger(stack: $stack, commitSha: $sha, runType: $type)"`
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		variables := map[string]interface{}{
			"stack": graphql.ID(row[1]),
			"sha":   (*graphql.String)(nil),
			"type":  structs.NewRunType("TRACKED"),
		}

		if err := authenticated.Client.Mutate(ctx, &mutation, variables); err != nil {
			return nil, err
		}

		return nil, nil
	default:
		return []table.Row{
			[]string{"Inspect", row[0]},
			[]string{"Trigger", row[0]},
		}, nil
	}
}

// Columns returns the columns of the worker pool table.
func (q *Stacks) Columns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 25},
		{Title: "Name", Width: 25},
		{Title: "Commit", Width: 15},
		{Title: "Author", Width: 15},
		{Title: "State", Width: 15},
	}
}

// Rows returns the rows of the worker pool table.
func (q *Stacks) Rows(ctx context.Context) (rows []table.Row, err error) {
	stacks, err := searchAllStacks(ctx, q.si)
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks {
		rows = append(rows, table.Row{
			stack.ID,
			stack.Name,
			cmd.HumanizeGitHash(stack.TrackedCommit.Hash),
			stack.TrackedCommit.AuthorName,
			stack.State,
		})
	}

	return rows, nil
}

type StackWatch struct {
	id string
}

func (w *StackWatch) Filtered(s string) error {
	return nil
}

func (s *StackWatch) Columns() []table.Column {
	return []table.Column{
		{Title: "Run ID", Width: 25},
		{Title: "Status", Width: 25},
		{Title: "Message", Width: 15},
		{Title: "Commit", Width: 15},
		{Title: "Triggered At", Width: 15},
		{Title: "Triggered By", Width: 15},
		{Title: "Changes", Width: 15},
		{Title: "Stack ID", Width: 15},
	}
}

func (s *StackWatch) Rows(ctx context.Context) (rows []table.Row, err error) {
	// TODO: make maxResults configurable
	runs, err := listRuns(ctx, s.id, 100)
	if err != nil {
		return nil, err
	}

	rows = append(rows, table.Row{"<-back", "", "", "", "", "", ""})
	for _, run := range runs {
		rows = append(rows, table.Row{
			run[0],
			run[1],
			run[2],
			run[3],
			run[4],
			run[5],
			run[6],
			run[7],
		})
	}

	return rows, nil
}

func (s *StackWatch) Selected(row table.Row) ([]table.Row, error) {
	ctx := context.Background()
	if row[0] == "<-back" {
		input := structs.SearchInput{
			OrderBy: &structs.QueryOrder{
				Field:     "starred",
				Direction: "DESC",
			},
		}

		st := &Stacks{input}
		t, err := draw.NewTable(ctx, st)
		if err != nil {
			return nil, err
		}

		return nil, t.DrawTable()
	}

	//return browser.OpenURL(authenticated.Client.URL("/stack/%s/run/%s", row[7], row[0]))
	//_, err := runLogsWithAction(context.Background(), row[7], row[0], nil)
	lines := make(chan string)
	go func() {
		getRunStates(ctx, row[7], row[0], lines, nil)
		close(lines)
	}()

	var logs strings.Builder
	for line := range lines {
		logs.WriteString(line)
	}
	p := tea.NewProgram(
		model{content: logs.String()},
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	_, err := p.Run()

	return nil, err
}

func listRuns(ctx context.Context, stackID string, maxResults int) ([][]string, error) {
	var results []runsTableQuery
	var before *string

	for len(results) < maxResults {
		var query struct {
			Stack *struct {
				Runs []runsTableQuery `graphql:"runs(before: $before)"`
			} `graphql:"stack(id: $stackId)"`
		}

		if err := authenticated.Client.Query(ctx, &query, map[string]interface{}{"stackId": stackID, "before": before}); err != nil {
			return nil, errors.Wrap(err, "failed to query run list")
		}

		if query.Stack == nil {
			return nil, fmt.Errorf("stack %q not found", stackID)
		}

		if len(query.Stack.Runs) == 0 {
			break
		}

		resultsToAdd := maxResults - len(results)
		if resultsToAdd > len(query.Stack.Runs) {
			resultsToAdd = len(query.Stack.Runs)
		}

		results = append(results, query.Stack.Runs[:resultsToAdd]...)

		before = &query.Stack.Runs[len(query.Stack.Runs)-1].ID
	}

	var tableData [][]string
	for _, run := range results {
		var deltaComponents []string

		if run.Delta.AddCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("+%d", run.Delta.AddCount))
		}
		if run.Delta.ChangeCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("~%d", run.Delta.ChangeCount))
		}
		if run.Delta.DeleteCount > 0 {
			deltaComponents = append(deltaComponents, fmt.Sprintf("-%d", run.Delta.DeleteCount))
		}

		delta := strings.Join(deltaComponents, " ")

		triggeredBy := run.TriggeredBy
		if triggeredBy == "" {
			triggeredBy = "Git commit"
		}

		createdAt := time.Unix(int64(run.CreatedAt), 0)

		tableData = append(tableData, []string{
			run.ID,
			run.State,
			run.Title,
			cmd.HumanizeGitHash(run.Commit.Hash),
			createdAt.Format(time.RFC3339),
			triggeredBy,
			delta,
			stackID,
		})
	}

	return tableData, nil
}

// RUN LOGS
var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type model struct {
	content  string
	ready    bool
	viewport viewport.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	title := titleStyle.Render("Run Logs \n(ctrl+c or q or esc to exit)")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// get rung states logs
func getRunStates(ctx context.Context, stack, run string, sink chan<- string, acFn actionOnRunState) (*structs.RunStateTransition, error) {
	var query struct {
		Stack *struct {
			Run *struct {
				History []structs.RunStateTransition `graphql:"history"`
			} `graphql:"run(id: $run)"`
		} `graphql:"stack(id: $stack)"`
	}

	variables := map[string]interface{}{
		"stack": graphql.ID(stack),
		"run":   graphql.ID(run),
	}

	reportedStates := make(map[structs.RunState]struct{})

	var backoff = time.Duration(0)

	for {
		if err := authenticated.Client.Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		if query.Stack == nil {
			return nil, fmt.Errorf("stack %q not found", stack)
		}

		if query.Stack.Run == nil {
			return nil, fmt.Errorf("run %q in stack %q not found", run, stack)
		}

		history := query.Stack.Run.History

		for index := range history {
			// Unlike the GUI, we go earliest first.
			transition := history[len(history)-index-1]

			if _, ok := reportedStates[transition.State]; ok {
				continue
			}
			backoff = 0
			reportedStates[transition.State] = struct{}{}

			if transition.HasLogs {
				if err := runStateLogs(ctx, stack, run, transition.State, transition.StateVersion, sink, transition.Terminal); err != nil {
					return nil, err
				}
			}

			if acFn != nil {
				if err := acFn(transition.State, stack, run); err != nil {
					return nil, fmt.Errorf("failed to execute action on run state: %w", err)
				}
			}

			if transition.Terminal {
				return &transition, nil
			}
		}

		time.Sleep(backoff * time.Second)

		if backoff < 5 {
			backoff++
		}
	}
}
