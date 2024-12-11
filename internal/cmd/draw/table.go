package draw

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Table is a table that can be drawn.
type Table struct {
	table     table.Model
	textInput textinput.Model

	td TableData

	width     int
	height    int
	baseStyle lipgloss.Style
	lastErr   error
}

type view string

const (
	viewTable view = "table"
	viewInput view = "input"
)

// TableData is the data for a table.
type TableData interface {
	// Columns returns the columns of the table.
	Columns() []table.Column

	// Rows returns the rows of the table.
	Rows(ctx context.Context) ([]table.Row, error)

	// Selected is called when a row is selected.
	// The entire row is passed to the function.
	Selected(table.Row) error

	Filtered(string) error
}

// NewTable creates a new table.
func NewTable(ctx context.Context, d TableData) (*Table, error) {
	rows, err := d.Rows(ctx)
	if err != nil {
		return nil, err
	}

	t := table.New(
		table.WithColumns(d.Columns()),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7C47FC")).
		Bold(false)
	t.SetStyles(s)

	bs := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("240"))

	ti := textinput.New()
	ti.Placeholder = "Enter text"
	ti.CharLimit = 156
	ti.Width = t.Width()

	width, height, err := term.GetSize(0)
	if err != nil {
		return nil, err
	}

	return &Table{
		table:     t,
		td:        d,
		textInput: ti,

		width:     width,
		height:    height,
		baseStyle: bs,

		lastErr: nil,
	}, nil
}

// DrawTable should be called to draw the table.
func (t *Table) DrawTable() error {
	if _, err := tea.NewProgram(t).Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}

// Init implements tea.Model.Init.
// Should not be called directly.
func (t Table) Init() tea.Cmd {
	return tickCmd(time.Second * 5)
}

// Update implements tea.Model.Update.
// Should not be called directly.
func (t Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+f":
			t.textInput.Focus()
			t.table.Blur()
		case "esc", "ctrl+c":
			if t.table.Focused() {
				return t, tea.Quit
			} else {
				t.table.Focus()

				t.textInput.Blur()
				t.textInput.Reset()
			}
		case "enter":
			if t.table.Focused() {
				err := t.td.Selected(t.table.SelectedRow())
				if err != nil {
					return t, t.saveErrorAndExit(err)
				}
			}
			if t.textInput.Focused() {
				err := t.td.Filtered(t.textInput.Value())
				if err != nil {
					return t, t.saveErrorAndExit(err)
				}
			}
			cmds = append(cmds, tickCmd(time.Microsecond))
		}
	case tickMsg:
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		rows, err := t.td.Rows(ctx)
		if err != nil {
			return t, t.saveErrorAndExit(err)
		}

		t.table.SetRows(rows)
		return t, tickCmd(time.Second * 5)
	}

	var cmd tea.Cmd
	t.textInput, cmd = t.textInput.Update(msg)
	cmds = append(cmds, cmd)

	t.table, cmd = t.table.Update(msg)
	cmds = append(cmds, cmd)

	return t, tea.Batch(cmds...)
}

// View implements tea.Model.View.
// Should not be called directly.
func (t Table) View() string {
	if t.lastErr != nil {
		return fmt.Sprintln("Exited with an error:", t.lastErr)
	}

	tableView := t.baseStyle.Render(t.table.View()) + "\n"

	filterText := "Filter stacks by name (CTRL+f to start)"
	if t.textInput.Focused() {
		filterText = "Filter stacks by name (CTRL+c to exit)"
	}

	filterView := fmt.Sprintf(
		"%s\n%s\n",
		filterText,
		t.textInput.View(),
	)

	views := []string{tableView, filterView}
	return lipgloss.Place(
		t.width, t.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Top, views...),
	)
}

func (t *Table) saveErrorAndExit(err error) tea.Cmd {
	t.lastErr = err
	return tea.Quit
}
