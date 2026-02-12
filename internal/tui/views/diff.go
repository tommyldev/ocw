package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
)

// Diff is the view for displaying git diff statistics
type Diff struct {
	instance   state.Instance
	viewport   viewport.Model
	width      int
	height     int
	diffStat   git.DiffStat
	diffFiles  []git.DiffFile
	gitManager *git.Git
	loading    bool
	err        error
}

// NewDiff creates a new Diff view
func NewDiff(instance state.Instance, gitManager *git.Git) *Diff {
	d := &Diff{
		instance:   instance,
		gitManager: gitManager,
		width:      80,
		height:     24,
		loading:    true,
	}

	// Initialize viewport
	d.viewport = viewport.New(d.width-4, d.height-8)
	d.viewport.YPosition = 3

	return d
}

// Init initializes the diff view
func (d *Diff) Init() tea.Cmd {
	return d.loadDiff()
}

// loadDiff loads the diff data
func (d *Diff) loadDiff() tea.Cmd {
	return func() tea.Msg {
		if d.gitManager == nil {
			return DiffLoadedMsg{Error: fmt.Errorf("git manager not available")}
		}

		// Get diff statistics
		stat, err := d.gitManager.DiffStatBranch(d.instance.Branch, d.instance.BaseBranch)
		if err != nil {
			return DiffLoadedMsg{Error: err}
		}

		// Get diff files
		files, err := d.gitManager.DiffFiles(fmt.Sprintf("%s..%s", d.instance.BaseBranch, d.instance.Branch))
		if err != nil {
			return DiffLoadedMsg{Error: err}
		}

		return DiffLoadedMsg{
			DiffStat:  stat,
			DiffFiles: files,
		}
	}
}

// DiffLoadedMsg is sent when diff data is loaded
type DiffLoadedMsg struct {
	DiffStat  git.DiffStat
	DiffFiles []git.DiffFile
	Error     error
}

// SetSize sets the size of the diff view
func (d *Diff) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width - 4
	d.viewport.Height = height - 8
}

// Update handles messages
func (d *Diff) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DiffLoadedMsg:
		d.loading = false
		if msg.Error != nil {
			d.err = msg.Error
			return d, nil
		}
		d.diffStat = msg.DiffStat
		d.diffFiles = msg.DiffFiles
		d.updateViewport()
		return d, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return d, nil // Signal to return to dashboard
		case "up", "k":
			d.viewport.LineUp(1)
		case "down", "j":
			d.viewport.LineDown(1)
		case "pgup":
			d.viewport.PageUp()
		case "pgdn":
			d.viewport.PageDown()
		case "home":
			d.viewport.GotoTop()
		case "end":
			d.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
		d.updateViewport()
	}

	return d, nil
}

// updateViewport updates the viewport content
func (d *Diff) updateViewport() {
	content := d.renderContent()
	d.viewport.SetContent(content)
}

// renderContent renders the diff content
func (d *Diff) renderContent() string {
	if d.loading {
		return "Loading diff..."
	}

	if d.err != nil {
		return fmt.Sprintf("Error: %v", d.err)
	}

	var sb strings.Builder

	// Render file list with status icons and colors
	for _, file := range d.diffFiles {
		icon := d.getStatusIcon(file.Status)
		color := d.getStatusColor(file.Status)
		styledIcon := color.Render(icon)
		sb.WriteString(fmt.Sprintf("%s %s\n", styledIcon, file.Path))
	}

	return sb.String()
}

// getStatusIcon returns the icon for a file status
func (d *Diff) getStatusIcon(status string) string {
	switch status {
	case "M":
		return "●"
	case "A":
		return "+"
	case "D":
		return "✕"
	case "R":
		return "→"
	default:
		return "?"
	}
}

// getStatusColor returns the color style for a file status
func (d *Diff) getStatusColor(status string) lipgloss.Style {
	switch status {
	case "M":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	case "A":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
	case "D":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	case "R":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // Blue
	default:
		return lipgloss.NewStyle()
	}
}

// View renders the diff view
func (d *Diff) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf("Diff: %s → %s", d.instance.Branch, d.instance.BaseBranch))

	summary := ""
	if !d.loading && d.err == nil {
		summary = summaryStyle.Render(d.diffStat.Summary)
	}

	viewportView := d.viewport.View()

	footer := footerStyle.Render("↑/k: up | ↓/j: down | PgUp/PgDn: page | Home/End: jump | ESC: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		summary,
		"",
		viewportView,
		"",
		footer,
	)
}
