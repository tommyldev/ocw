package views

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"strings"

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

type StatusStyles struct {
	Active   lipgloss.Style
	Idle     lipgloss.Style
	Paused   lipgloss.Style
	Error    lipgloss.Style
	Merged   lipgloss.Style
	Done     lipgloss.Style
	Conflict lipgloss.Style
}

type CustomDelegate struct {
	statusStyles StatusStyles
	allInstances []state.Instance
}

func NewCustomDelegate(statusStyles StatusStyles, allInstances []state.Instance) *CustomDelegate {
	return &CustomDelegate{statusStyles: statusStyles, allInstances: allInstances}
}

// Height returns the height of a list item
func (d *CustomDelegate) Height() int {
	return 2
}

// Spacing returns the spacing between list items
func (d *CustomDelegate) Spacing() int {
	return 1
}

// Update handles messages for the delegate
func (d *CustomDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d *CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(InstanceItem)
	if !ok {
		return
	}

	inst := i.instance

	statusIcon := d.getStatusIcon(inst.Status)
	statusStyle := d.getStatusStyle(inst.Status)

	elapsed := time.Since(inst.CreatedAt)
	elapsedStr := formatDuration(elapsed)

	subTermCount := len(inst.SubTerminals)
	subTermStr := fmt.Sprintf("%d sub-terms", subTermCount)
	if subTermCount > 0 {
		labeledCount := 0
		for _, st := range inst.SubTerminals {
			if st.Label != "" {
				labeledCount++
			}
		}
		if labeledCount > 0 {
			subTermStr = fmt.Sprintf("%d sub-terms (%d labeled)", subTermCount, labeledCount)
		}
	}

	conflictStr := ""
	if len(inst.ConflictsWith) > 0 {
		conflictStr = " ⚠"
	}

	depStr := ""
	if len(inst.DependsOn) > 0 {
		nameMap := make(map[string]string)
		for _, ai := range d.allInstances {
			nameMap[ai.ID] = ai.Name
		}
		var depNames []string
		for _, depID := range inst.DependsOn {
			if name, ok := nameMap[depID]; ok {
				depNames = append(depNames, name)
			}
		}
		if len(depNames) > 0 {
			depStr = fmt.Sprintf(" → depends on %s", strings.Join(depNames, ", "))
		}
	}

	firstLine := fmt.Sprintf("%d. %s %s | %s | %s%s%s",
		index+1,
		statusStyle.Render(statusIcon),
		inst.Name,
		elapsedStr,
		subTermStr,
		conflictStr,
		depStr,
	)

	secondLine := fmt.Sprintf("   Branch: %s → %s",
		inst.Branch,
		inst.BaseBranch,
	)

	fmt.Fprintf(w, "%s\n%s", firstLine, secondLine)
}

// getStatusIcon returns the icon for a given status
func (d *CustomDelegate) getStatusIcon(status string) string {
	switch status {
	case "running":
		return "●"
	case "idle":
		return "○"
	case "paused":
		return "⏸"
	case "error":
		return "✗"
	case "merged":
		return "✓"
	case "done":
		return "✔"
	default:
		return "?"
	}
}

func (d *CustomDelegate) getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return d.statusStyles.Active
	case "idle":
		return d.statusStyles.Idle
	case "paused":
		return d.statusStyles.Paused
	case "error":
		return d.statusStyles.Error
	case "merged":
		return d.statusStyles.Merged
	case "done":
		return d.statusStyles.Done
	default:
		return d.statusStyles.Idle
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

type Dashboard struct {
	list           list.Model
	instances      []state.Instance
	width          int
	height         int
	statusStyles   StatusStyles
	lastRefresh    time.Time
	lastConflict   time.Time
	selectedIndex  int
	refreshTicker  *time.Ticker
	conflictTicker *time.Ticker
}

func NewDashboard(instances []state.Instance, statusStyles StatusStyles) *Dashboard {
	d := &Dashboard{
		instances:     instances,
		width:         80,
		height:        24,
		statusStyles:  statusStyles,
		lastRefresh:   time.Now(),
		lastConflict:  time.Now(),
		selectedIndex: 0,
	}

	items := make([]list.Item, len(instances))
	for i, inst := range instances {
		items[i] = InstanceItem{instance: inst}
	}

	delegate := NewCustomDelegate(statusStyles, instances)
	l := list.New(items, delegate, 80, 20)
	l.Title = "OCW Instances"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	d.list = l

	return d
}

func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.list.SetWidth(width - 4)
	d.list.SetHeight(height - 6)
}

func (d *Dashboard) GetSelectedIndex() int {
	return d.list.Index()
}

func (d *Dashboard) Init() tea.Cmd {
	d.refreshTicker = time.NewTicker(1500 * time.Millisecond)
	d.conflictTicker = time.NewTicker(30 * time.Second)
	return tea.Batch(
		d.tickRefresh(),
		d.tickConflict(),
	)
}

type RefreshMsg struct{}
type ConflictCheckMsg struct{}

func (d *Dashboard) tickRefresh() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}

func (d *Dashboard) tickConflict() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return ConflictCheckMsg{}
	})
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefreshMsg:
		d.lastRefresh = time.Now()
		items := make([]list.Item, len(d.instances))
		for i, inst := range d.instances {
			items[i] = InstanceItem{instance: inst}
		}
		d.list.SetItems(items)
		return d, d.tickRefresh()

	case ConflictCheckMsg:
		d.lastConflict = time.Now()
		return d, d.tickConflict()

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			d.list.CursorUp()
		case "down", "j":
			d.list.CursorDown()
		}

	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	d.list, cmd = d.list.Update(msg)
	return d, cmd
}

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
