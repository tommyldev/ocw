package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tui/views"
)

// AppState represents the current state of the application
type AppState string

const (
	StateDashboard     AppState = "dashboard"
	StateDetail        AppState = "detail"
	StateCreate        AppState = "create"
	StateHelp          AppState = "help"
	StateDiff          AppState = "diff"
	StateMerge         AppState = "merge"
	StateDeleteConfirm AppState = "delete-confirm"
	StateLog           AppState = "log"
	StateSendPrompt    AppState = "send-prompt"
)

// FocusMsg is sent when user wants to focus on an instance
type FocusMsg struct {
	InstanceIndex int
}

// FocusCompleteMsg is sent after focus returns to dashboard
type FocusCompleteMsg struct {
	Error error
}

// DeleteMsg is sent when deletion completes
type DeleteMsg struct {
	Success bool
	Error   error
}

// SendPromptMsg is sent when prompt sending completes
type SendPromptMsg struct {
	Success bool
	Error   error
}

// App is the root Bubbletea model
type App struct {
	ctx                *Context
	state              AppState
	keyMap             KeyMap
	styles             Styles
	instances          []state.Instance
	selected           int
	width              int
	height             int
	dashboard          *views.Dashboard
	create             *views.Create
	diff               *views.Diff
	merge              *views.Merge
	help               *views.Help
	log                *views.Log
	err                error
	program            *tea.Program
	deleteInstanceID   string
	deleteInstanceName string
	promptInstanceID   string
	promptText         string
	promptFeedback     string
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

	helpStyles := views.HelpStyles{
		Title:     app.styles.Header,
		Header:    app.styles.SelectedItem,
		Border:    app.styles.FocusedBorder,
		Text:      lipgloss.NewStyle(),
		Highlight: app.styles.SelectedItem,
		Footer:    app.styles.Footer,
	}
	app.help = views.NewHelp(helpStyles)
	app.help.SetSize(app.width, app.height)

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
		if a.merge != nil {
			a.merge.SetSize(msg.Width, msg.Height)
		}
		if a.help != nil {
			a.help.SetSize(msg.Width, msg.Height)
		}
		if a.log != nil {
			a.log.SetSize(msg.Width, msg.Height)
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
	case DeleteMsg:
		if msg.Error != nil {
			a.err = msg.Error
			a.state = StateDashboard
			return a, nil
		}
		a.state = StateDashboard
		return a.refreshInstances()
	case SendPromptMsg:
		if msg.Error != nil {
			a.err = msg.Error
		} else {
			a.promptFeedback = "Prompt sent successfully!"
		}
		a.state = StateDashboard
		return a, nil
	case views.DiffLoadedMsg:
		// Handle diff loaded message
		if a.diff != nil {
			model, cmd := a.diff.Update(msg)
			a.diff = model.(*views.Diff)
			return a, cmd
		}
		if a.merge != nil {
			model, cmd := a.merge.Update(msg)
			a.merge = model.(*views.Merge)
			return a, cmd
		}
	case views.LogLoadedMsg:
		if a.log != nil {
			model, cmd := a.log.Update(msg)
			a.log = model.(*views.Log)
			return a, cmd
		}
	case views.MergeMsg:
		if msg.Error != nil {
			a.err = msg.Error
			a.state = StateDashboard
			return a, nil
		}
		a.state = StateDashboard
		return a.refreshInstances()
	case views.MergeConflictCheckMsg:
		if a.merge != nil {
			model, cmd := a.merge.Update(msg)
			a.merge = model.(*views.Merge)
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
	case StateMerge:
		if a.merge != nil {
			model, cmd := a.merge.Update(msg)
			a.merge = model.(*views.Merge)
			// Check if ESC was pressed to return to dashboard
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
				a.state = StateDashboard
				return a, nil
			}
			return a, cmd
		}
	case StateLog:
		if a.log != nil {
			model, cmd := a.log.Update(msg)
			a.log = model.(*views.Log)
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
	case StateMerge:
		if a.merge != nil {
			return a.merge.View()
		}
		return "Loading..."
	case StateHelp:
		if a.help != nil {
			return a.help.View()
		}
		return "Loading..."
	case StateLog:
		if a.log != nil {
			return a.log.View()
		}
		return "Loading..."
	case StateDeleteConfirm:
		return a.renderDeleteConfirm()
	case StateSendPrompt:
		return a.renderSendPrompt()
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
	case "n", "N":
		if a.state == StateDeleteConfirm {
			a.state = StateDashboard
			return a, nil
		}
		if a.state == StateDashboard {
			a.state = StateCreate
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
	case "d":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				selectedInstance := a.instances[selectedIdx]
				a.deleteInstanceID = selectedInstance.ID
				a.deleteInstanceName = selectedInstance.Name
				a.state = StateDeleteConfirm
				return a, nil
			}
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
	case "m":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				selectedInstance := a.instances[selectedIdx]
				gitMgr := git.NewGit(selectedInstance.WorktreePath)
				mergeStyles := views.MergeStyles{
					Title:      a.styles.Header,
					Error:      a.styles.ErrorText,
					Success:    a.styles.StatusMergedStyle,
					Warning:    a.styles.ConflictWarning,
					Help:       a.styles.Footer,
					FormBorder: a.styles.FocusedBorder,
				}
				a.merge = views.NewMerge(selectedInstance, a.ctx.Manager, gitMgr, mergeStyles)
				a.merge.SetSize(a.width, a.height)
				a.state = StateMerge
				return a, a.merge.Init()
			}
		}
	case "l":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				selectedInstance := a.instances[selectedIdx]
				a.log = views.NewLog(selectedInstance, a.ctx.Manager.Tmux())
				a.log.SetSize(a.width, a.height)
				a.state = StateLog
				return a, a.log.Init()
			}
		}
	case "s":
		if a.state == StateDashboard && a.dashboard != nil {
			selectedIdx := a.dashboard.GetSelectedIndex()
			if selectedIdx >= 0 && selectedIdx < len(a.instances) {
				selectedInstance := a.instances[selectedIdx]
				a.promptInstanceID = selectedInstance.ID
				a.promptText = ""
				a.promptFeedback = ""
				a.state = StateSendPrompt
				return a, nil
			}
		}
	case "enter":
		if a.state == StateSendPrompt {
			if a.promptText != "" {
				return a, a.sendPromptCmd(a.promptInstanceID, a.promptText)
			}
			a.state = StateDashboard
			return a, nil
		}
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
	case "backspace":
		if a.state == StateSendPrompt {
			if len(a.promptText) > 0 {
				a.promptText = a.promptText[:len(a.promptText)-1]
			}
			return a, nil
		}
	case "esc":
		if a.state == StateCreate {
			a.state = StateDashboard
			return a, nil
		}
		if a.state == StateDeleteConfirm {
			a.state = StateDashboard
			return a, nil
		}
		if a.state == StateHelp {
			a.state = StateDashboard
			return a, nil
		}
		if a.state == StateSendPrompt {
			a.state = StateDashboard
			return a, nil
		}
	case "y":
		if a.state == StateDeleteConfirm {
			return a, a.deleteInstanceCmd(a.deleteInstanceID)
		}
	}

	// Handle text input for send prompt
	if a.state == StateSendPrompt {
		key := msg.String()
		if len(key) == 1 && key >= " " && key <= "~" {
			a.promptText += key
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
	case StateHelp:
		if a.help != nil {
			model, cmd := a.help.Update(msg)
			a.help = model.(*views.Help)
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

func (a *App) deleteInstanceCmd(instanceID string) tea.Cmd {
	return func() tea.Msg {
		if a.ctx.Manager == nil {
			return DeleteMsg{Success: false, Error: fmt.Errorf("manager not available")}
		}

		err := a.ctx.Manager.DeleteInstance(instanceID, false, false)
		if err != nil {
			return DeleteMsg{Success: false, Error: fmt.Errorf("failed to delete instance: %w", err)}
		}

		return DeleteMsg{Success: true, Error: nil}
	}
}

func (a *App) renderDeleteConfirm() string {
	title := a.styles.Header.Render("Delete Instance")
	instanceName := a.styles.FocusedBorder.Render(a.deleteInstanceName)
	warning := a.styles.ErrorText.Render("⚠ This will remove the worktree and kill all processes")
	prompt := "Delete instance? [y/N] (ESC to cancel)"

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", title, instanceName, warning, prompt)
}

func (a *App) sendPromptCmd(instanceID, promptText string) tea.Cmd {
	return func() tea.Msg {
		if a.ctx.Manager == nil {
			return SendPromptMsg{Success: false, Error: fmt.Errorf("manager not available")}
		}

		var instance *state.Instance
		for i := range a.instances {
			if a.instances[i].ID == instanceID {
				instance = &a.instances[i]
				break
			}
		}

		if instance == nil {
			return SendPromptMsg{Success: false, Error: fmt.Errorf("instance not found")}
		}

		if err := a.ctx.Manager.Tmux().SendKeys(instance.PrimaryPane, promptText); err != nil {
			return SendPromptMsg{Success: false, Error: fmt.Errorf("failed to send prompt: %w", err)}
		}

		return SendPromptMsg{Success: true, Error: nil}
	}
}

func (a *App) renderSendPrompt() string {
	title := a.styles.Header.Render("Send Prompt to Instance")
	textBox := a.styles.FocusedBorder.Render(a.promptText + "█")
	help := a.styles.Footer.Render("Type your prompt | Enter: Send | ESC: Cancel")
	feedback := ""
	if a.promptFeedback != "" {
		feedback = "\n\n" + a.styles.StatusActiveStyle.Render(a.promptFeedback)
	}
	return fmt.Sprintf("%s\n\n%s\n\n%s%s", title, textBox, help, feedback)
}
