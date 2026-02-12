package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/state"
)

// InstanceItem wraps state.Instance for list rendering
type InstanceItem struct {
	instance state.Instance
}

func (i InstanceItem) FilterValue() string {
	return i.instance.Name
}

func (i InstanceItem) Title() string {
	return i.instance.Name
}

func (i InstanceItem) Description() string {
	elapsed := time.Since(i.instance.CreatedAt).String()
	return fmt.Sprintf("Branch: %s | Status: %s | Created: %s ago",
		i.instance.Branch, i.instance.Status, elapsed)
}

// Dashboard is the main dashboard view
type Dashboard struct {
	list      list.Model
	instances []state.Instance
	width     int
	height    int
}

// NewDashboard creates a new Dashboard view
func NewDashboard(instances []state.Instance) *Dashboard {
	d := &Dashboard{
		instances: instances,
		width:     80,
		height:    24,
	}

	// Create list items from instances
	items := make([]list.Item, len(instances))
	for i, inst := range instances {
		items[i] = InstanceItem{instance: inst}
	}

	// Create list model
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "OCW Instances"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	d.list = l

	return d
}

// SetSize sets the size of the dashboard
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.list.SetWidth(width - 4)
	d.list.SetHeight(height - 6)
}

// Init initializes the dashboard
func (d *Dashboard) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			d.list.CursorUp()
		case "down", "j":
			d.list.CursorDown()
		case "enter":
			// Handle selection
			return d, nil
		}
	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	d.list, cmd = d.list.Update(msg)
	return d, cmd
}

// View renders the dashboard
func (d *Dashboard) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	header := headerStyle.Render("OCW - Open Code Workspace")

	listView := d.list.View()

	footer := footerStyle.Render(
		fmt.Sprintf("Total instances: %d | Press ? for help | Press q to quit", len(d.instances)),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		listView,
		"",
		footer,
	)
}
