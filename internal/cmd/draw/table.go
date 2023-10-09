package draw

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Table is a table that can be drawn.
type Table struct {
	table table.Model
	td    TableData

	width     int
	height    int
	baseStyle lipgloss.Style

	lastErr error
}

// TableData is the data for a table.
type TableData interface {
	// Columns returns the columns of the table.
	Columns() []table.Column

	// Rows returns the rows of the table.
	Rows(ctx context.Context) ([]table.Row, error)

	// Selected is called when a row is selected.
	// The entire row is passed to the function.
	Selected(table.Row) error
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

	width, height, err := term.GetSize(0)
	if err != nil {
		return nil, err
	}

	return &Table{
		table: t,
		td:    d,

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
	return tickCmd()
}

// Update implements tea.Model.Update.
// Should not be called directly.
func (t Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			return t, tea.Quit
		case "enter":
			err := t.td.Selected(t.table.SelectedRow())
			if err != nil {
				return t, t.saveErrorAndExit(err)
			}
		}
	case tickMsg:
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		rows, err := t.td.Rows(ctx)
		if err != nil {
			return t, t.saveErrorAndExit(err)
		}

		t.table.SetRows(rows)
		return t, tickCmd()
	}

	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// View implements tea.Model.View.
// Should not be called directly.
func (t Table) View() string {
	if t.lastErr != nil {
		return fmt.Sprintln("Exited with an error:", t.lastErr)
	}

	return lipgloss.Place(
		t.width, t.height,
		lipgloss.Center, lipgloss.Center,
		t.baseStyle.Render(t.table.View())+"\n",
	)
}

func (t *Table) saveErrorAndExit(err error) tea.Cmd {
	t.lastErr = err
	return tea.Quit
}
