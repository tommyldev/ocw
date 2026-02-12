package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tmux"
)

// Log is the view for displaying pane scrollback history
type Log struct {
	instance state.Instance
	viewport viewport.Model
	width    int
	height   int
	tmux     *tmux.Tmux
	loading  bool
	err      error
}

// NewLog creates a new Log view
func NewLog(instance state.Instance, tmuxClient *tmux.Tmux) *Log {
	l := &Log{
		instance: instance,
		tmux:     tmuxClient,
		width:    80,
		height:   24,
		loading:  true,
	}

	// Initialize viewport
	l.viewport = viewport.New(l.width-4, l.height-8)
	l.viewport.YPosition = 3

	return l
}

// Init initializes the log view
func (l *Log) Init() tea.Cmd {
	return l.loadLogs()
}

// loadLogs loads the pane scrollback content
func (l *Log) loadLogs() tea.Cmd {
	return func() tea.Msg {
		if l.tmux == nil {
			return LogLoadedMsg{Error: fmt.Errorf("tmux client not available")}
		}

		target := fmt.Sprintf("%s.0", l.instance.TmuxWindow)
		output, err := l.tmux.CapturePaneScrollback(target)
		if err != nil {
			return LogLoadedMsg{Error: err}
		}

		return LogLoadedMsg{
			Content: output,
		}
	}
}

// LogLoadedMsg is sent when log data is loaded
type LogLoadedMsg struct {
	Content string
	Error   error
}

// SetSize sets the size of the log view
func (l *Log) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.viewport.Width = width - 4
	l.viewport.Height = height - 8
}

// Update handles messages
func (l *Log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LogLoadedMsg:
		l.loading = false
		if msg.Error != nil {
			l.err = msg.Error
			return l, nil
		}
		l.viewport.SetContent(msg.Content)
		// Auto-scroll to bottom
		l.viewport.GotoBottom()
		return l, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return l, nil // Signal to return to dashboard
		case "up", "k":
			l.viewport.LineUp(1)
		case "down", "j":
			l.viewport.LineDown(1)
		case "pgup":
			l.viewport.PageUp()
		case "pgdn":
			l.viewport.PageDown()
		case "home":
			l.viewport.GotoTop()
		case "end":
			l.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		l.SetSize(msg.Width, msg.Height)
	}

	return l, nil
}

// View renders the log view
func (l *Log) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf("Logs: %s", l.instance.Name))

	var content string
	if l.loading {
		content = "Loading logs..."
	} else if l.err != nil {
		content = fmt.Sprintf("Error: %v", l.err)
	} else {
		content = l.viewport.View()
	}

	footer := footerStyle.Render("↑/k: up | ↓/j: down | PgUp/PgDn: page | Home/End: jump | ESC: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}
