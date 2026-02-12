package views

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/workspace"
)

// CreateMsg is sent when instance creation completes
type CreateMsg struct {
	Instance *state.Instance
	Error    error
}

// Create is the view for creating new instances
type Create struct {
	form          *huh.Form
	branchName    string
	baseBranch    string
	manager       *workspace.Manager
	defaultBase   string
	width         int
	height        int
	creating      bool
	creationError string
	styles        CreateStyles
}

// CreateStyles holds styling for the create view
type CreateStyles struct {
	Title      lipgloss.Style
	Error      lipgloss.Style
	Help       lipgloss.Style
	FormBorder lipgloss.Style
}

// NewCreate creates a new Create view
func NewCreate(manager *workspace.Manager, defaultBase string, styles CreateStyles) *Create {
	c := &Create{
		manager:     manager,
		defaultBase: defaultBase,
		width:       80,
		height:      24,
		styles:      styles,
	}

	c.buildForm()
	return c
}

// Init initializes the create view
func (c *Create) Init() tea.Cmd {
	return nil
}

// buildForm constructs the huh form
func (c *Create) buildForm() {
	c.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Branch Name").
				Placeholder("feature/my-feature").
				Value(&c.branchName).
				Validate(c.validateBranchName),
			huh.NewInput().
				Title("Base Branch").
				Placeholder(c.defaultBase).
				Value(&c.baseBranch).
				Validate(c.validateBaseBranch),
		),
	).
		WithTheme(huh.ThemeCatppuccin()).
		WithShowHelp(true).
		WithShowErrors(true)
}

// validateBranchName validates the branch name input
func (c *Create) validateBranchName(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for spaces
	if strings.Contains(s, " ") {
		return fmt.Errorf("branch name cannot contain spaces")
	}

	// Check for valid git ref characters
	// Allow: alphanumeric, hyphens, underscores, slashes, periods
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)
	if !validPattern.MatchString(s) {
		return fmt.Errorf("branch name contains invalid characters (only alphanumeric, /, -, _, . allowed)")
	}

	// Check for leading/trailing slashes or dots
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") ||
		strings.HasSuffix(s, "/") || strings.HasSuffix(s, ".") {
		return fmt.Errorf("branch name cannot start or end with / or .")
	}

	return nil
}

// validateBaseBranch validates the base branch input
func (c *Create) validateBaseBranch(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("base branch cannot be empty")
	}

	// Check for spaces
	if strings.Contains(s, " ") {
		return fmt.Errorf("base branch cannot contain spaces")
	}

	return nil
}

// Update handles messages
func (c *Create) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel and return to dashboard
			return c, nil
		}

	case CreateMsg:
		if msg.Error != nil {
			c.creating = false
			c.creationError = msg.Error.Error()
			return c, nil
		}
		// Success - return to dashboard
		return c, nil
	}

	// Delegate to form
	form, cmd := c.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		c.form = f

		// Check if form was submitted
		if c.form.State == huh.StateCompleted {
			c.creating = true
			c.creationError = ""

			// Use default base branch if not provided
			baseBranch := c.baseBranch
			if strings.TrimSpace(baseBranch) == "" {
				baseBranch = c.defaultBase
			}

			// Create instance in background
			return c, c.createInstanceCmd(c.branchName, baseBranch)
		}
	}

	return c, cmd
}

// createInstanceCmd creates an instance asynchronously
func (c *Create) createInstanceCmd(branchName, baseBranch string) tea.Cmd {
	return func() tea.Msg {
		if c.manager == nil {
			return CreateMsg{Error: fmt.Errorf("manager not available")}
		}

		opts := workspace.CreateOpts{
			Name:       branchName,
			Branch:     branchName,
			BaseBranch: baseBranch,
		}

		instance, err := c.manager.CreateInstance(opts)
		return CreateMsg{Instance: instance, Error: err}
	}
}

// View renders the create view
func (c *Create) View() string {
	if c.creating {
		return c.renderSpinner()
	}

	if c.creationError != "" {
		return c.renderError()
	}

	title := c.styles.Title.Render("Create New Instance")
	help := c.styles.Help.Render("Press esc to cancel")

	formView := c.form.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		formView,
		"",
		help,
	)
}

// renderSpinner renders a loading spinner during creation
func (c *Create) renderSpinner() string {
	spinner := "⠋ Creating instance..."
	title := c.styles.Title.Render("Create New Instance")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		spinner,
	)
}

// renderError renders the error message
func (c *Create) renderError() string {
	title := c.styles.Title.Render("Create New Instance")
	errorMsg := c.styles.Error.Render(fmt.Sprintf("Error: %s", c.creationError))
	help := c.styles.Help.Render("Press esc to go back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		errorMsg,
		"",
		help,
	)
}

// SetSize sets the size of the view
func (c *Create) SetSize(width, height int) {
	c.width = width
	c.height = height
	if c.form != nil {
		c.form.WithWidth(width - 4)
	}
}
