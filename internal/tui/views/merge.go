package views

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tmux"
	"github.com/tommyzliu/ocw/internal/workspace"
)

// MergeMsg is sent when merge operation completes
type MergeMsg struct {
	PRURL string
	Error error
}

// MergeConflictCheckMsg is sent when conflict check completes
type MergeConflictCheckMsg struct {
	HasConflicts  bool
	ConflictFiles []string
	Error         error
}

// MergeDependencyCheckMsg is sent when dependency check completes
type MergeDependencyCheckMsg struct {
	UnmergedDeps []string
	Error        error
}

// ResolveConflictsRequestMsg requests conflict resolution flow
type ResolveConflictsRequestMsg struct {
	InstanceID string
}

// Merge is the view for merging branches and creating PRs
type Merge struct {
	instance          state.Instance
	manager           *workspace.Manager
	conflictDetector  *workspace.ConflictDetector
	gitManager        *git.Git
	tmuxClient        *tmux.Tmux
	form              *huh.Form
	prTitle           string
	prBody            string
	width             int
	height            int
	diffStat          git.DiffStat
	diffFiles         []git.DiffFile
	hasConflicts      bool
	conflictFiles     []string
	conflictCheckDone bool
	unmergedDeps      []string
	depCheckDone      bool
	merging           bool
	mergeError        string
	prURL             string
	allInstances      []state.Instance
	styles            MergeStyles
}

// MergeStyles holds styling for the merge view
type MergeStyles struct {
	Title      lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Help       lipgloss.Style
	FormBorder lipgloss.Style
}

// NewMerge creates a new Merge view
func NewMerge(instance state.Instance, manager *workspace.Manager, gitManager *git.Git, tmuxClient *tmux.Tmux, allInstances []state.Instance, styles MergeStyles) *Merge {
	m := &Merge{
		instance:          instance,
		manager:           manager,
		gitManager:        gitManager,
		tmuxClient:        tmuxClient,
		conflictDetector:  workspace.NewConflictDetector(gitManager),
		allInstances:      allInstances,
		width:             80,
		height:            24,
		styles:            styles,
		conflictCheckDone: false,
		depCheckDone:      false,
		merging:           false,
		// Default PR title: format branch name
		prTitle: formatBranchNameForPR(instance.Branch),
		prBody:  generatePRDescriptionFromActivity(instance, tmuxClient),
	}

	m.buildForm()
	return m
}

// formatBranchNameForPR converts a branch name like "feature/add-auth" to "Add auth"
func formatBranchNameForPR(branch string) string {
	// Remove common prefixes
	branch = strings.TrimPrefix(branch, "feature/")
	branch = strings.TrimPrefix(branch, "feat/")
	branch = strings.TrimPrefix(branch, "bugfix/")
	branch = strings.TrimPrefix(branch, "fix/")
	branch = strings.TrimPrefix(branch, "hotfix/")

	// Replace hyphens and underscores with spaces
	branch = strings.ReplaceAll(branch, "-", " ")
	branch = strings.ReplaceAll(branch, "_", " ")

	// Capitalize first letter
	if len(branch) > 0 {
		branch = strings.ToUpper(branch[:1]) + branch[1:]
	}

	return branch
}

func generatePRDescriptionFromActivity(instance state.Instance, tmuxClient *tmux.Tmux) string {
	if tmuxClient == nil || instance.TmuxWindow == "" {
		return ""
	}

	openCodePaneTarget := fmt.Sprintf("%s.0", instance.TmuxWindow)
	scrollback, err := tmuxClient.CapturePaneScrollback(openCodePaneTarget)
	if err != nil {
		return ""
	}

	activities := parseScrollbackActivity(scrollback)
	if len(activities) == 0 {
		return ""
	}

	var description strings.Builder
	description.WriteString("## Changes\n")
	description.WriteString("Auto-generated from OpenCode activity:\n\n")

	for _, activity := range activities {
		description.WriteString(fmt.Sprintf("- %s\n", activity))
	}

	return description.String()
}

func parseScrollbackActivity(scrollback string) []string {
	if scrollback == "" {
		return nil
	}

	lines := strings.Split(scrollback, "\n")
	activities := []string{}
	seenActivities := make(map[string]bool)
	maxActivities := 10

	meaningfulPatterns := struct {
		fileChanges *regexp.Regexp
		errorFixes  *regexp.Regexp
		actions     *regexp.Regexp
	}{
		fileChanges: regexp.MustCompile(`(?i)(created|modified|updated|added|deleted|fixed|removed)\s+[\w/.-]+\.(go|js|ts|py|rb|java|css|html|md|yml|yaml|json|txt|sh)`),
		errorFixes:  regexp.MustCompile(`(?i)(fixed|resolved|corrected)\s+.*?(error|bug|issue|problem)`),
		actions:     regexp.MustCompile(`(?i)(implemented|added|created|built|configured|setup|installed|integrated)`),
	}

	noisePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*$`),
		regexp.MustCompile(`^[\d:/-]+\s*$`),
		regexp.MustCompile(`^(\x1b\[[0-9;]*m)*\s*$`),
		regexp.MustCompile(`(?i)^(ls|cd|pwd|cat|echo)`),
		regexp.MustCompile(`^[>$#]\s*$`),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if isNoiseLine(line, noisePatterns) {
			continue
		}

		activity := extractActivity(line, meaningfulPatterns)
		if activity == "" {
			continue
		}

		normalizedActivity := normalizeActivity(activity)
		if len(normalizedActivity) > 10 && !seenActivities[normalizedActivity] {
			seenActivities[normalizedActivity] = true
			activities = append(activities, normalizedActivity)

			if len(activities) >= maxActivities {
				break
			}
		}
	}

	return activities
}

func isNoiseLine(line string, noisePatterns []*regexp.Regexp) bool {
	for _, pattern := range noisePatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func extractActivity(line string, patterns struct {
	fileChanges *regexp.Regexp
	errorFixes  *regexp.Regexp
	actions     *regexp.Regexp
}) string {
	if patterns.fileChanges.MatchString(line) {
		return patterns.fileChanges.FindString(line)
	}

	if patterns.errorFixes.MatchString(line) {
		return patterns.errorFixes.FindString(line)
	}

	if patterns.actions.MatchString(line) && len(line) > 30 {
		return line
	}

	return ""
}

func normalizeActivity(activity string) string {
	activity = strings.TrimSpace(activity)
	activity = strings.Trim(activity, ".,;:!?")
	return activity
}

// Init initializes the merge view
func (m *Merge) Init() tea.Cmd {
	return tea.Batch(
		m.loadDiff(),
		m.checkConflicts(),
		m.checkDependencies(),
	)
}

func (m *Merge) checkDependencies() tea.Cmd {
	return func() tea.Msg {
		if m.manager == nil {
			return MergeDependencyCheckMsg{Error: fmt.Errorf("manager not available")}
		}

		unmerged, err := m.manager.CheckDependenciesMerged(m.instance.ID)
		if err != nil {
			return MergeDependencyCheckMsg{Error: err}
		}

		return MergeDependencyCheckMsg{UnmergedDeps: unmerged}
	}
}

// loadDiff loads the diff data
func (m *Merge) loadDiff() tea.Cmd {
	return func() tea.Msg {
		if m.gitManager == nil {
			return DiffLoadedMsg{Error: fmt.Errorf("git manager not available")}
		}

		// Get diff statistics
		stat, err := m.gitManager.DiffStatBranch(m.instance.Branch, m.instance.BaseBranch)
		if err != nil {
			return DiffLoadedMsg{Error: err}
		}

		// Get diff files
		files, err := m.gitManager.DiffFiles(fmt.Sprintf("%s..%s", m.instance.BaseBranch, m.instance.Branch))
		if err != nil {
			return DiffLoadedMsg{Error: err}
		}

		return DiffLoadedMsg{
			DiffStat:  stat,
			DiffFiles: files,
		}
	}
}

// checkConflicts checks for merge conflicts
func (m *Merge) checkConflicts() tea.Cmd {
	return func() tea.Msg {
		if m.conflictDetector == nil {
			return MergeConflictCheckMsg{Error: fmt.Errorf("conflict detector not available")}
		}

		hasConflicts, conflictFiles, err := m.conflictDetector.CheckMergeConflicts(m.instance)
		if err != nil {
			return MergeConflictCheckMsg{Error: err}
		}

		return MergeConflictCheckMsg{
			HasConflicts:  hasConflicts,
			ConflictFiles: conflictFiles,
		}
	}
}

// buildForm constructs the huh form
func (m *Merge) buildForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("PR Title").
				Placeholder("Brief description of changes").
				Value(&m.prTitle).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("PR title cannot be empty")
					}
					return nil
				}),
			huh.NewText().
				Title("PR Body (optional)").
				Placeholder("Detailed description...").
				Value(&m.prBody).
				CharLimit(5000),
		),
	).
		WithTheme(huh.ThemeCatppuccin()).
		WithShowHelp(true).
		WithShowErrors(true)
}

// SetSize sets the size of the merge view
func (m *Merge) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.form != nil {
		m.form.WithWidth(width - 4)
	}
}

func (m *Merge) GetConflictFiles() []string {
	return m.conflictFiles
}

// Update handles messages
func (m *Merge) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DiffLoadedMsg:
		if msg.Error != nil {
			m.mergeError = msg.Error.Error()
			return m, nil
		}
		m.diffStat = msg.DiffStat
		m.diffFiles = msg.DiffFiles
		return m, nil

	case MergeConflictCheckMsg:
		m.conflictCheckDone = true
		if msg.Error != nil {
			m.mergeError = msg.Error.Error()
			return m, nil
		}
		m.hasConflicts = msg.HasConflicts
		m.conflictFiles = msg.ConflictFiles
		return m, nil

	case MergeDependencyCheckMsg:
		m.depCheckDone = true
		if msg.Error != nil {
			m.mergeError = msg.Error.Error()
			return m, nil
		}
		m.unmergedDeps = msg.UnmergedDeps
		return m, nil

	case MergeMsg:
		m.merging = false
		if msg.Error != nil {
			m.mergeError = msg.Error.Error()
			return m, nil
		}
		m.prURL = msg.PRURL
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, nil
		case "c":
			if m.hasConflicts {
				return m, func() tea.Msg {
					return ResolveConflictsRequestMsg{InstanceID: m.instance.ID}
				}
			}
		}
	}

	// Delegate to form if not merging and conflicts are checked
	if !m.merging && m.conflictCheckDone && m.depCheckDone && !m.hasConflicts && len(m.unmergedDeps) == 0 && m.prURL == "" {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f

			// Check if form was submitted
			if m.form.State == huh.StateCompleted {
				m.merging = true
				m.mergeError = ""
				return m, m.performMerge()
			}
		}
		return m, cmd
	}

	return m, nil
}

// performMerge executes the merge operation
func (m *Merge) performMerge() tea.Cmd {
	return func() tea.Msg {
		if m.manager == nil {
			return MergeMsg{Error: fmt.Errorf("manager not available")}
		}

		// Push branch
		if err := m.manager.PushBranch(m.instance.ID); err != nil {
			return MergeMsg{Error: fmt.Errorf("failed to push branch: %w", err)}
		}

		// Create PR
		prURL, err := m.manager.CreatePR(m.instance.ID, m.prTitle, m.prBody)
		if err != nil {
			return MergeMsg{Error: fmt.Errorf("branch pushed but PR creation failed: %w", err)}
		}

		return MergeMsg{PRURL: prURL}
	}
}

// View renders the merge view
func (m *Merge) View() string {
	if m.merging {
		return m.renderMerging()
	}

	if m.prURL != "" {
		return m.renderSuccess()
	}

	if m.mergeError != "" {
		return m.renderError()
	}

	if !m.conflictCheckDone || !m.depCheckDone {
		return m.renderLoading()
	}

	if m.hasConflicts {
		return m.renderConflicts()
	}

	if len(m.unmergedDeps) > 0 {
		return m.renderUnmergedDeps()
	}

	return m.renderForm()
}

// renderLoading renders a loading state
func (m *Merge) renderLoading() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))
	spinner := "⠋ Checking conflicts and dependencies..."
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		spinner,
	)
}

// renderConflicts renders the conflict warning
func (m *Merge) renderConflicts() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))

	warning := m.styles.Warning.Render("⚠ Cannot merge: conflicts detected")

	var conflictList strings.Builder
	conflictList.WriteString("\nConflicting files:\n")
	for _, file := range m.conflictFiles {
		conflictList.WriteString(fmt.Sprintf("  • %s\n", file))
	}

	help := m.styles.Help.Render("Press 'c' to resolve conflicts | ESC to go back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		warning,
		conflictList.String(),
		"",
		help,
	)
}

func (m *Merge) renderUnmergedDeps() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))

	warning := m.styles.Warning.Render("⚠ Cannot merge: unmerged dependencies")

	nameMap := make(map[string]string)
	for _, inst := range m.allInstances {
		nameMap[inst.ID] = inst.Name
	}

	var depList strings.Builder
	depList.WriteString("\nMerge these first:\n")
	for _, depID := range m.unmergedDeps {
		name := depID
		if n, ok := nameMap[depID]; ok {
			name = n
		}
		depList.WriteString(fmt.Sprintf("  • %s (%s)\n", name, depID))
	}

	help := m.styles.Help.Render("ESC to go back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		warning,
		depList.String(),
		"",
		help,
	)
}

func (m *Merge) renderForm() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))

	// Diff summary
	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)
	summary := summaryStyle.Render(m.diffStat.Summary)

	// File list (first 10 files)
	var fileList strings.Builder
	maxFiles := 10
	for i, file := range m.diffFiles {
		if i >= maxFiles {
			remaining := len(m.diffFiles) - maxFiles
			fileList.WriteString(fmt.Sprintf("  ... and %d more files\n", remaining))
			break
		}
		icon := getStatusIcon(file.Status)
		color := getStatusColor(file.Status)
		styledIcon := color.Render(icon)
		fileList.WriteString(fmt.Sprintf("  %s %s\n", styledIcon, file.Path))
	}

	// Conflict status
	conflictStatus := m.styles.Success.Render("✓ No conflicts")
	depStatus := m.styles.Success.Render("✓ Dependencies satisfied")

	formView := m.form.View()

	help := m.styles.Help.Render("Press Enter to push & create PR | ESC to cancel")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		summary,
		"",
		fileList.String(),
		"",
		conflictStatus,
		depStatus,
		"",
		formView,
		"",
		help,
	)
}

// renderMerging renders the merging state
func (m *Merge) renderMerging() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))
	spinner := "⠋ Pushing branch and creating PR..."
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		spinner,
	)
}

// renderSuccess renders the success state
func (m *Merge) renderSuccess() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))

	success := m.styles.Success.Render("✓ Pull request created successfully!")

	prURLText := fmt.Sprintf("PR URL: %s", m.prURL)

	help := m.styles.Help.Render("Press ESC to return to dashboard")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		success,
		"",
		prURLText,
		"",
		help,
	)
}

// renderError renders the error state
func (m *Merge) renderError() string {
	title := m.styles.Title.Render(fmt.Sprintf("Merge: %s → %s", m.instance.Branch, m.instance.BaseBranch))
	errorMsg := m.styles.Error.Render(fmt.Sprintf("Error: %s", m.mergeError))
	help := m.styles.Help.Render("Press ESC to go back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		errorMsg,
		"",
		help,
	)
}

// getStatusIcon returns the icon for a file status
func getStatusIcon(status string) string {
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
func getStatusColor(status string) lipgloss.Style {
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
