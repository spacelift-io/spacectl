package draw

import (
	"context"
	"fmt"
	"strings"
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

	choices []table.Row
	cursor  int
}

// TableData is the data for a table.
type TableData interface {
	// Columns returns the columns of the table.
	Columns() []table.Column

	// Rows returns the rows of the table.
	Rows(ctx context.Context) ([]table.Row, error)

	// Selected is called when a row is selected.
	// The entire row is passed to the function.
	Selected(table.Row) ([]table.Row, error)

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

	screen := &Table{
		table:     t,
		td:        d,
		textInput: ti,

		width:     width,
		height:    height,
		baseStyle: bs,

		lastErr: nil,
	}

	return screen, nil
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

func (t Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if t.choices != nil {
		return t.choicesUpdate(msg)
	}

	return t.tableUpdate(msg)
}

func (t Table) choicesUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			t.choices = nil
			t.cursor = 0

			return t, nil
		case "enter":
			opts, err := t.td.Selected(t.choices[t.cursor])
			if err != nil {
				return t, t.saveErrorAndExit(err)
			}

			t.choices = opts
			t.cursor = 0
		case "down", "j":
			t.cursor++
			if t.cursor >= len(t.choices) {
				t.cursor = 0
			}

		case "up", "k":
			t.cursor--
			if t.cursor < 0 {
				t.cursor = len(t.choices) - 1
			}
		}
	default:
		// OK
	}

	return t, nil
}

// Update implements tea.Model.Update.
// Should not be called directly.
func (t Table) tableUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				opts, err := t.td.Selected(t.table.SelectedRow())
				if err != nil {
					return t, t.saveErrorAndExit(err)
				}

				t.choices = opts
				t.cursor = 0
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
		return t, tickCmd(time.Second * 3)
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

	var views []string
	if t.choices != nil {
		views = t.choicesView()
	} else {
		views = t.tableView()
	}

	return lipgloss.Place(
		t.width, t.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Top, views...),
	)
}

func (t Table) tableView() []string {
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

	return []string{tableView, filterView}
}

func (t Table) choicesView() []string {
	s := strings.Builder{}
	s.WriteString("What how to proceed?\n\n")

	for i := 0; i < len(t.choices); i++ {
		if t.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}

		s.WriteString(t.choices[i][0])
		s.WriteString("\n")
	}
	s.WriteString("\n(press CTRL+C to go back)\n")

	return []string{s.String()}
}

func (t *Table) saveErrorAndExit(err error) tea.Cmd {
	t.lastErr = err
	return tea.Quit
}
