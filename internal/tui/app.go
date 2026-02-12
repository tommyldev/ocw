package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tui/views"
)

// AppState represents the current state of the application
type AppState string

const (
	StateDashboard AppState = "dashboard"
	StateDetail    AppState = "detail"
	StateHelp      AppState = "help"
)

// App is the root Bubbletea model
type App struct {
	ctx       *Context
	state     AppState
	keyMap    KeyMap
	styles    Styles
	instances []state.Instance
	selected  int
	width     int
	height    int
	dashboard *views.Dashboard
	err       error
}

// NewApp creates a new App instance
func NewApp(ctx *Context) *App {
	app := &App{
		ctx:       ctx,
		state:     StateDashboard,
		keyMap:    DefaultKeyMap(),
		styles:    DefaultStyles(),
		instances: []state.Instance{},
		selected:  0,
		width:     80,
		height:    24,
		err:       nil,
	}

	// Load instances from state
	if ctx.Manager != nil {
		stateData, err := ctx.Manager.Store().Load()
		if err == nil && stateData != nil {
			app.instances = stateData.Instances
		}
	}

	// Initialize dashboard view
	app.dashboard = views.NewDashboard(app.instances)

	return app
}

// Init initializes the app
func (a *App) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ctx.Width = msg.Width
		a.ctx.Height = msg.Height
		if a.dashboard != nil {
			a.dashboard.SetSize(msg.Width, msg.Height)
		}
		return a, nil
	}

	// Delegate to current view
	switch a.state {
	case StateDashboard:
		if a.dashboard != nil {
			model, cmd := a.dashboard.Update(msg)
			a.dashboard = model.(*views.Dashboard)
			return a, cmd
		}
	}

	return a, nil
}

// View renders the app
func (a *App) View() string {
	if a.err != nil {
		return a.styles.ErrorText.Render(fmt.Sprintf("Error: %v", a.err))
	}

	switch a.state {
	case StateDashboard:
		if a.dashboard != nil {
			return a.dashboard.View()
		}
		return "Loading..."
	case StateHelp:
		return a.renderHelp()
	default:
		return "Unknown state"
	}
}

// handleKeyMsg handles key messages
func (a *App) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return a, tea.Quit
	case "?":
		if a.state == StateDashboard {
			a.state = StateHelp
		} else {
			a.state = StateDashboard
		}
		return a, nil
	case "r":
		if a.state == StateDashboard {
			return a.refreshInstances()
		}
	}

	// Delegate to current view
	switch a.state {
	case StateDashboard:
		if a.dashboard != nil {
			model, cmd := a.dashboard.Update(msg)
			a.dashboard = model.(*views.Dashboard)
			return a, cmd
		}
	}

	return a, nil
}

// refreshInstances reloads instances from state
func (a *App) refreshInstances() (tea.Model, tea.Cmd) {
	if a.ctx.Manager != nil {
		stateData, err := a.ctx.Manager.Store().Load()
		if err != nil {
			a.err = err
			return a, nil
		}
		if stateData != nil {
			a.instances = stateData.Instances
			a.dashboard = views.NewDashboard(a.instances)
			a.dashboard.SetSize(a.width, a.height)
		}
	}
	return a, nil
}

// renderHelp renders the help screen
func (a *App) renderHelp() string {
	help := "OCW - Open Code Workspace\n\n"
	help += "Key Bindings:\n"
	help += fmt.Sprintf("  %s - %s\n", "↑/k", "move up")
	help += fmt.Sprintf("  %s - %s\n", "↓/j", "move down")
	help += fmt.Sprintf("  %s - %s\n", "enter", "select")
	help += fmt.Sprintf("  %s - %s\n", "n", "new instance")
	help += fmt.Sprintf("  %s - %s\n", "d", "delete instance")
	help += fmt.Sprintf("  %s - %s\n", "r", "refresh")
	help += fmt.Sprintf("  %s - %s\n", "?", "toggle help")
	help += fmt.Sprintf("  %s - %s\n", "ctrl+c/q", "quit")
	help += "\nPress ? to close help"
	return help
}
