package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all lipgloss styles for the TUI
type Styles struct {
	// Status icons
	StatusRunning string
	StatusStopped string
	StatusError   string
	StatusPending string

	// Colors
	FocusedBorder  lipgloss.Style
	BlurredBorder  lipgloss.Style
	SelectedItem   lipgloss.Style
	UnselectedItem lipgloss.Style
	Header         lipgloss.Style
	Footer         lipgloss.Style
	ErrorText      lipgloss.Style
	SuccessText    lipgloss.Style
	WarningText    lipgloss.Style
	InfoText       lipgloss.Style
}

// DefaultStyles returns the default style configuration
func DefaultStyles() Styles {
	return Styles{
		// Status icons
		StatusRunning: "●",
		StatusStopped: "○",
		StatusError:   "✗",
		StatusPending: "◐",

		// Focused border
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")),

		// Blurred border
		BlurredBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),

		// Selected item
		SelectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true),

		// Unselected item
		UnselectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),

		// Header
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1),

		// Footer
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1),

		// Error text
		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		// Success text
		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")),

		// Warning text
		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")),

		// Info text
		InfoText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")),
	}
}
