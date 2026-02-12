package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tui/views"
)

// AppState represents the current state of the application
type AppState string

const (
	StateDashboard AppState = "dashboard"
	StateDetail    AppState = "detail"
	StateCreate    AppState = "create"
	StateHelp      AppState = "help"
	StateDiff      AppState = "diff"
)

// FocusMsg is sent when user wants to focus on an instance
type FocusMsg struct {
	InstanceIndex int
}

// FocusCompleteMsg is sent after focus returns to dashboard
type FocusCompleteMsg struct {
	Error error
}

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
	create    *views.Create
	diff      *views.Diff
	err       error
	program   *tea.Program
}

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
		program:   nil,
	}

	if ctx.Manager != nil {
		stateData, err := ctx.Manager.Store().Load()
		if err == nil && stateData != nil {
			app.instances = stateData.Instances
		}
	}

	statusStyles := views.StatusStyles{
		Active:   app.styles.StatusActiveStyle,
		Idle:     app.styles.StatusIdleStyle,
		Paused:   app.styles.StatusPausedStyle,
		Error:    app.styles.StatusErrorStyle,
		Merged:   app.styles.StatusMergedStyle,
		Done:     app.styles.StatusDoneStyle,
		Conflict: app.styles.ConflictWarning,
	}

	app.dashboard = views.NewDashboard(app.instances, statusStyles)

	// Initialize create view
	createStyles := views.CreateStyles{
		Title:      app.styles.Header,
		Error:      app.styles.ErrorText,
		Help:       app.styles.Footer,
		FormBorder: app.styles.FocusedBorder,
	}
	defaultBase := "main"
	if ctx.Config != nil && ctx.Config.Workspace.BaseBranch != "" {
		defaultBase = ctx.Config.Workspace.BaseBranch
	}
	app.create = views.NewCreate(ctx.Manager, defaultBase, createStyles)

	return app
}

// SetProgram sets the tea.Program reference for terminal control
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
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
		if a.create != nil {
			a.create.SetSize(msg.Width, msg.Height)
		}
		if a.diff != nil {
			a.diff.SetSize(msg.Width, msg.Height)
		}
		return a, nil
	case views.CreateMsg:
		// Handle create completion
		if msg.Error != nil {
			a.err = msg.Error
			return a, nil
		}
		// Success - return to dashboard
		a.state = StateDashboard
		return a.refreshInstances()
	case views.DiffLoadedMsg:
		// Handle diff loaded message
		if a.diff != nil {
			model, cmd := a.diff.Update(msg)
			a.diff = model.(*views.Diff)
			return a, cmd
		}
	case FocusCompleteMsg:
		if msg.Error != nil {
			a.err = msg.Error
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
	case StateCreate:
		if a.create != nil {
			model, cmd := a.create.Update(msg)
			a.create = model.(*views.Create)
			return a, cmd
		}
	case StateDiff:
		if a.diff != nil {
			model, cmd := a.diff.Update(msg)
			a.diff = model.(*views.Diff)
			// Check if ESC was pressed to return to dashboard
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
				a.state = StateDashboard
				return a, nil
			}
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
	case StateCreate:
		if a.create != nil {
			return a.create.View()
		}
		return "Loading..."
	case StateDiff:
		if a.diff != nil {
			return a.diff.View()
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
		} else if a.state == StateCreate {
			a.state = StateDashboard
		} else {
			a.state = StateDashboard
		}
		return a, nil
	case "r":
		if a.state == StateDashboard {
			return a.refreshInstances()
		}
	case "n":
		if a.state == StateDashboard {
			a.state = StateCreate
			// Reset create form
			defaultBase := "main"
			if a.ctx.Config != nil && a.ctx.Config.Workspace.BaseBranch != "" {
				defaultBase = a.ctx.Config.Workspace.BaseBranch
			}
			createStyles := views.CreateStyles{
				Title:      a.styles.Header,
				Error:      a.styles.ErrorText,
				Help:       a.styles.Footer,
				FormBorder: a.styles.FocusedBorder,
			}
			a.create = views.NewCreate(a.ctx.Manager, defaultBase, createStyles)
			a.create.SetSize(a.width, a.height)
			return a, nil
		}
	case "f":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				selectedInstance := a.instances[selectedIdx]
				gitMgr := git.NewGit(selectedInstance.WorktreePath)
				a.diff = views.NewDiff(selectedInstance, gitMgr)
				a.diff.SetSize(a.width, a.height)
				a.state = StateDiff
				return a, a.diff.Init()
			}
		}
	case "enter":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				return a, a.focusInstance(selectedIdx)
			}
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if a.state == StateDashboard {
			idx := int(msg.String()[0] - '0' - 1)
			if idx >= 0 && idx < len(a.instances) {
				return a, a.focusInstance(idx)
			}
		}
	case "esc":
		if a.state == StateCreate {
			a.state = StateDashboard
			return a, nil
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
	case StateCreate:
		if a.create != nil {
			model, cmd := a.create.Update(msg)
			a.create = model.(*views.Create)
			return a, cmd
		}
	}

	return a, nil
}

func (a *App) refreshInstances() (tea.Model, tea.Cmd) {
	if a.ctx.Manager != nil {
		stateData, err := a.ctx.Manager.Store().Load()
		if err != nil {
			a.err = err
			return a, nil
		}
		if stateData != nil {
			a.instances = stateData.Instances
			statusStyles := views.StatusStyles{
				Active:   a.styles.StatusActiveStyle,
				Idle:     a.styles.StatusIdleStyle,
				Paused:   a.styles.StatusPausedStyle,
				Error:    a.styles.StatusErrorStyle,
				Merged:   a.styles.StatusMergedStyle,
				Done:     a.styles.StatusDoneStyle,
				Conflict: a.styles.ConflictWarning,
			}
			a.dashboard = views.NewDashboard(a.instances, statusStyles)
			a.dashboard.SetSize(a.width, a.height)
		}
	}
	return a, nil
}

func (a *App) focusInstance(instanceIndex int) tea.Cmd {
	return func() tea.Msg {
		if instanceIndex < 0 || instanceIndex >= len(a.instances) {
			return FocusCompleteMsg{Error: fmt.Errorf("invalid instance index")}
		}

		instance := a.instances[instanceIndex]

		if a.ctx.Manager == nil {
			return FocusCompleteMsg{Error: fmt.Errorf("manager not available")}
		}

		sessionName, err := a.ctx.Manager.EnsureSession()
		if err != nil {
			return FocusCompleteMsg{Error: fmt.Errorf("failed to get session name: %w", err)}
		}

		if a.program == nil {
			return FocusCompleteMsg{Error: fmt.Errorf("program not available")}
		}

		if err := a.program.ReleaseTerminal(); err != nil {
			return FocusCompleteMsg{Error: fmt.Errorf("failed to release terminal: %w", err)}
		}

		err = a.ctx.Manager.Tmux().AttachWindow(sessionName, instance.TmuxWindow)

		if restoreErr := a.program.RestoreTerminal(); restoreErr != nil {
			return FocusCompleteMsg{Error: fmt.Errorf("failed to restore terminal: %w", restoreErr)}
		}

		return FocusCompleteMsg{Error: err}
	}
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
