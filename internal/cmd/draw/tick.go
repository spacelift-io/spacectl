package draw

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
