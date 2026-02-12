package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Help represents the help view
type Help struct {
	width  int
	height int
	styles HelpStyles
	scroll int
}

// HelpStyles holds styles for the help view
type HelpStyles struct {
	Title     lipgloss.Style
	Header    lipgloss.Style
	Border    lipgloss.Style
	Text      lipgloss.Style
	Highlight lipgloss.Style
	Footer    lipgloss.Style
}

// NewHelp creates a new help view
func NewHelp(styles HelpStyles) *Help {
	return &Help{
		width:  80,
		height: 24,
		styles: styles,
		scroll: 0,
	}
}

// SetSize sets the size of the help view
func (h *Help) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// Init initializes the help view
func (h *Help) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (h *Help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?":
			// Return to dashboard
			return h, nil
		case "j", "down":
			h.scroll++
		case "k", "up":
			if h.scroll > 0 {
				h.scroll--
			}
		case "g":
			h.scroll = 0
		case "G":
			h.scroll = 999 // Will be clamped by View()
		}
	}
	return h, nil
}

// View renders the help view
func (h *Help) View() string {
	content := h.buildContent()
	lines := strings.Split(content, "\n")

	// Calculate max scroll
	maxScroll := 0
	if len(lines) > h.height-4 {
		maxScroll = len(lines) - (h.height - 4)
	}
	if h.scroll > maxScroll {
		h.scroll = maxScroll
	}

	// Get visible lines
	endIdx := h.scroll + (h.height - 4)
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	visibleLines := lines[h.scroll:endIdx]

	// Build output
	output := h.styles.Title.Render("OCW - Open Code Workspace Help") + "\n"
	output += strings.Repeat("─", h.width) + "\n\n"
	output += strings.Join(visibleLines, "\n")

	// Add footer
	footer := fmt.Sprintf("  [j/k or ↑/↓] scroll  [g/G] top/bottom  [?/esc] close  [%d/%d]",
		h.scroll+1, len(lines))
	output += "\n\n" + h.styles.Footer.Render(footer)

	return output
}

// buildContent builds the help content
func (h *Help) buildContent() string {
	var sb strings.Builder

	// Dashboard hotkeys
	sb.WriteString(h.styles.Header.Render("DASHBOARD HOTKEYS") + "\n")
	sb.WriteString(h.buildTable([][]string{
		{"Key", "Action"},
		{"n", "Create new instance"},
		{"d", "Delete selected instance"},
		{"f", "Show diff for selected instance"},
		{"m", "Merge selected instance"},
		{"t", "Show sub-terminals for selected instance"},
		{"r", "Refresh instances"},
		{"enter", "Focus on selected instance"},
		{"1-9", "Quick focus on instance 1-9"},
	}))
	sb.WriteString("\n")

	// Navigation hotkeys
	sb.WriteString(h.styles.Header.Render("NAVIGATION HOTKEYS") + "\n")
	sb.WriteString(h.buildTable([][]string{
		{"Key", "Action"},
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"←/h", "Move left"},
		{"→/l", "Move right"},
		{"esc", "Return to previous view"},
	}))
	sb.WriteString("\n")

	// Global hotkeys
	sb.WriteString(h.styles.Header.Render("GLOBAL HOTKEYS") + "\n")
	sb.WriteString(h.buildTable([][]string{
		{"Key", "Action"},
		{"?", "Toggle help"},
		{"q", "Quit"},
		{"ctrl+c", "Quit"},
	}))
	sb.WriteString("\n")

	// Help view hotkeys
	sb.WriteString(h.styles.Header.Render("HELP VIEW HOTKEYS") + "\n")
	sb.WriteString(h.buildTable([][]string{
		{"Key", "Action"},
		{"j/↓", "Scroll down"},
		{"k/↑", "Scroll up"},
		{"g", "Jump to top"},
		{"G", "Jump to bottom"},
		{"?/esc", "Close help"},
	}))
	sb.WriteString("\n")

	// Tips
	sb.WriteString(h.styles.Header.Render("TIPS") + "\n")
	sb.WriteString("• Use keyboard shortcuts to navigate quickly\n")
	sb.WriteString("• Press 'enter' to focus on a workspace and attach to its tmux window\n")
	sb.WriteString("• Use 'f' to view changes before merging with 'm'\n")
	sb.WriteString("• Create sub-terminals within a workspace for running tests/servers\n")
	sb.WriteString("• OCW automatically detects conflicts between workspace changes\n")

	return sb.String()
}

// buildTable builds a formatted table
func (h *Help) buildTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// Header row
	headerRow := rows[0]
	for i, cell := range headerRow {
		sb.WriteString("  ")
		sb.WriteString(h.styles.Highlight.Render(padRight(cell, colWidths[i])))
		if i < len(headerRow)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Separator
	totalWidth := 0
	for i, width := range colWidths {
		totalWidth += width + 2
		if i < len(colWidths)-1 {
			totalWidth += 2
		}
	}
	sb.WriteString(strings.Repeat("─", totalWidth))
	sb.WriteString("\n")

	// Data rows
	for _, row := range rows[1:] {
		for i, cell := range row {
			sb.WriteString("  ")
			sb.WriteString(padRight(cell, colWidths[i]))
			if i < len(row)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// padRight pads a string to the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
