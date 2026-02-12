package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/tommyzliu/ocw/internal/state"
)

// CreateOpts contains options for creating a new instance.
type CreateOpts struct {
	Name        string // Display name for the instance
	Branch      string // Branch name to create/use
	BaseBranch  string // Base branch to branch from
	InitCommand string // Command to run after creating worktree
}

// InstanceStatus represents the current status of an instance.
type InstanceStatus struct {
	Instance  state.Instance
	PIDAlive  bool
	PaneDead  bool
	IsRunning bool
	CanResume bool
	CanPause  bool
}

// CreateInstance creates a new OCW instance with a dedicated worktree and tmux window.
// Steps:
// 1. Validate branch name and check if branch exists
// 2. Create git worktree at sanitized path
// 3. Create tmux window in the session
// 4. Set remain-on-exit for the primary pane
// 5. Launch opencode command and capture PID
// 6. Register instance in state
func (m *Manager) CreateInstance(opts CreateOpts) (*state.Instance, error) {
	if opts.Branch == "" {
		return nil, fmt.Errorf("branch name cannot be empty")
	}

	if err := m.checkNestedWorktree(); err != nil {
		return nil, err
	}

	id, err := state.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate instance ID: %w", err)
	}

	sanitizedBranch := sanitizeBranchName(opts.Branch)
	worktreePath := filepath.Join(m.repoRoot, m.config.Workspace.WorktreeDir, sanitizedBranch)

	sessionName, err := m.EnsureSession()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	branchExists := m.git.BranchExists(opts.Branch)

	if branchExists {
		if err := m.git.WorktreeAddExisting(worktreePath, opts.Branch); err != nil {
			return nil, fmt.Errorf("failed to create worktree for existing branch %q: %w\n\nThe branch exists but couldn't be checked out. This may happen if:\n  - The branch is already checked out in another worktree\n  - There are filesystem permission issues\n\nTo fix:\n  1. Check existing worktrees: git worktree list\n  2. Remove stale worktrees if needed: git worktree prune", opts.Branch, err)
		}
	} else {
		baseBranch := opts.BaseBranch
		if baseBranch == "" {
			baseBranch = m.config.Workspace.BaseBranch
		}

		if err := m.git.WorktreeAdd(worktreePath, opts.Branch, baseBranch); err != nil {
			return nil, fmt.Errorf("failed to create worktree with new branch %q from %q: %w\n\nTo fix:\n  1. Ensure base branch %q exists: git branch -a | grep %s\n  2. Fetch latest changes: git fetch\n  3. Check disk space: df -h", opts.Branch, baseBranch, err, baseBranch, baseBranch)
		}
	}

	// Create tmux window for the instance
	windowName := opts.Name
	if windowName == "" {
		windowName = opts.Branch
	}

	windowID, err := m.tmux.NewWindow(sessionName, windowName, worktreePath)
	if err != nil {
		// Cleanup worktree on failure
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Set remain-on-exit for the window so we can detect when opencode exits
	if err := m.tmux.SetRemainOnExit(windowID, true); err != nil {
		// Cleanup on failure
		_ = m.tmux.KillWindow(windowID)
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to set remain-on-exit: %w", err)
	}

	// Get the primary pane ID (the window's first pane)
	panes, err := m.tmux.ListPanes(windowID)
	if err != nil || len(panes) == 0 {
		// Cleanup on failure
		_ = m.tmux.KillWindow(windowID)
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to get primary pane: %w", err)
	}
	primaryPaneID := panes[0].ID

	// Build opencode command
	opencodeCmd := m.buildOpencodeCommand()

	// Launch opencode in the window
	if err := m.tmux.SendKeys(windowID, opencodeCmd); err != nil {
		// Cleanup on failure
		_ = m.tmux.KillWindow(windowID)
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to launch opencode: %w", err)
	}

	// Wait a moment for the process to start, then capture PID
	time.Sleep(100 * time.Millisecond)
	panes, err = m.tmux.ListPanes(windowID)
	if err != nil || len(panes) == 0 {
		// Cleanup on failure
		_ = m.tmux.KillWindow(windowID)
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to capture PID: %w", err)
	}
	pid := panes[0].PID

	// Run init command if specified
	if opts.InitCommand != "" {
		if err := m.tmux.SendKeys(windowID, opts.InitCommand); err != nil {
			// Cleanup on failure
			_ = m.tmux.KillWindow(windowID)
			_ = m.git.WorktreeRemove(worktreePath, true)
			return nil, fmt.Errorf("failed to run init command: %w", err)
		}
	}

	// Create instance record
	now := time.Now()
	instance := state.Instance{
		ID:            id,
		Name:          windowName,
		Branch:        opts.Branch,
		BaseBranch:    opts.BaseBranch,
		WorktreePath:  worktreePath,
		TmuxWindow:    windowID,
		PrimaryPane:   primaryPaneID,
		SubTerminals:  []state.SubTerminal{},
		PID:           pid,
		Status:        "running",
		CreatedAt:     now,
		LastActivity:  now,
		ConflictsWith: []string{},
	}

	// Save to state
	if err := m.store.AddInstance(instance); err != nil {
		// Cleanup on failure
		_ = m.tmux.KillWindow(windowID)
		_ = m.git.WorktreeRemove(worktreePath, true)
		return nil, fmt.Errorf("failed to save instance to state: %w", err)
	}

	return &instance, nil
}

// DeleteInstance removes an instance and cleans up all associated resources.
// Steps:
// 1. Load instance from state
// 2. Kill sub-terminals
// 3. Kill tmux window
// 4. Remove worktree
// 5. Optionally delete branch
// 6. Remove from state
func (m *Manager) DeleteInstance(id string, force bool, deleteBranch bool) error {
	// Load state
	st, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find the instance
	var instance *state.Instance
	for i := range st.Instances {
		if st.Instances[i].ID == id {
			instance = &st.Instances[i]
			break
		}
	}

	if instance == nil {
		return fmt.Errorf("instance %q not found", id)
	}

	// Kill sub-terminals first
	for _, subTerm := range instance.SubTerminals {
		_ = m.tmux.KillPane(subTerm.PaneID)
	}

	// Kill the tmux window
	if err := m.tmux.KillWindow(instance.TmuxWindow); err != nil && !force {
		return fmt.Errorf("failed to kill tmux window: %w", err)
	}

	// Remove the worktree
	if err := m.git.WorktreeRemove(instance.WorktreePath, force); err != nil {
		if !force {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}

	// Optionally delete the branch (only if requested and not the base branch)
	if deleteBranch && instance.Branch != instance.BaseBranch && instance.Branch != "" {
		if err := m.git.BranchDelete(instance.Branch, true); err != nil {
			return fmt.Errorf("failed to delete branch: %w", err)
		}
	}

	// Remove from state
	if err := m.store.RemoveInstance(id); err != nil {
		return fmt.Errorf("failed to remove instance from state: %w", err)
	}

	return nil
}

// PauseInstance pauses an instance by sending SIGSTOP to its main process.
func (m *Manager) PauseInstance(id string) error {
	// Load state
	st, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find the instance
	var instance *state.Instance
	for i := range st.Instances {
		if st.Instances[i].ID == id {
			instance = &st.Instances[i]
			break
		}
	}

	if instance == nil {
		return fmt.Errorf("instance %q not found", id)
	}

	// Check if process is alive
	if !isProcessAlive(instance.PID) {
		return fmt.Errorf("process with PID %d is not running", instance.PID)
	}

	// Send SIGSTOP
	if err := syscall.Kill(instance.PID, syscall.SIGSTOP); err != nil {
		return fmt.Errorf("failed to pause process: %w", err)
	}

	// Update status
	if err := m.store.UpdateInstance(id, func(inst *state.Instance) {
		inst.Status = "paused"
	}); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}

// ResumeInstance resumes a paused instance by sending SIGCONT to its main process.
func (m *Manager) ResumeInstance(id string) error {
	// Load state
	st, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find the instance
	var instance *state.Instance
	for i := range st.Instances {
		if st.Instances[i].ID == id {
			instance = &st.Instances[i]
			break
		}
	}

	if instance == nil {
		return fmt.Errorf("instance %q not found", id)
	}

	// Send SIGCONT
	if err := syscall.Kill(instance.PID, syscall.SIGCONT); err != nil {
		return fmt.Errorf("failed to resume process: %w", err)
	}

	// Update status
	if err := m.store.UpdateInstance(id, func(inst *state.Instance) {
		inst.Status = "running"
		inst.LastActivity = time.Now()
	}); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}

// GetInstanceStatus retrieves the current status of an instance.
func (m *Manager) GetInstanceStatus(id string) (*InstanceStatus, error) {
	// Load state
	st, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Find the instance
	var instance *state.Instance
	for i := range st.Instances {
		if st.Instances[i].ID == id {
			instance = &st.Instances[i]
			break
		}
	}

	if instance == nil {
		return nil, fmt.Errorf("instance %q not found", id)
	}

	// Check if PID is alive
	pidAlive := isProcessAlive(instance.PID)

	// Check if pane is dead
	panes, err := m.tmux.ListPanes(instance.TmuxWindow)
	paneDead := err != nil || len(panes) == 0 || panes[0].Dead

	// Determine running state
	isRunning := pidAlive && !paneDead

	status := &InstanceStatus{
		Instance:  *instance,
		PIDAlive:  pidAlive,
		PaneDead:  paneDead,
		IsRunning: isRunning,
		CanResume: instance.Status == "paused" && pidAlive,
		CanPause:  instance.Status == "running" && pidAlive,
	}

	return status, nil
}

// ListInstances returns all instances from state.
func (m *Manager) ListInstances() ([]state.Instance, error) {
	st, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return st.Instances, nil
}

// GetInstance retrieves a specific instance by ID.
func (m *Manager) GetInstance(id string) (*state.Instance, error) {
	st, err := m.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	for i := range st.Instances {
		if st.Instances[i].ID == id {
			return &st.Instances[i], nil
		}
	}

	return nil, fmt.Errorf("instance %q not found", id)
}

// RenameInstance updates the display name of an instance.
func (m *Manager) RenameInstance(id, newName string) error {
	if newName == "" {
		return fmt.Errorf("new name cannot be empty")
	}

	st, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	var instance *state.Instance
	for i := range st.Instances {
		if st.Instances[i].ID == id {
			instance = &st.Instances[i]
			break
		}
	}

	if instance == nil {
		return fmt.Errorf("instance %q not found", id)
	}

	instance.Name = newName

	if err := m.store.Save(st); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// sanitizeBranchName converts a branch name to a safe filesystem path.
// Handles slashes, special characters, and ensures valid directory names.
func sanitizeBranchName(branch string) string {
	// Replace slashes with hyphens (e.g., feature/foo -> feature-foo)
	sanitized := strings.ReplaceAll(branch, "/", "-")

	// Remove or replace other problematic characters
	// Only allow alphanumeric, hyphens, underscores, and periods
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	// Remove leading/trailing hyphens or periods (invalid directory names)
	sanitized = strings.Trim(sanitized, "-.")

	// Ensure not empty after sanitization
	if sanitized == "" {
		sanitized = "branch"
	}

	return sanitized
}

// isProcessAlive checks if a process with the given PID is running.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	// This doesn't actually send a signal, just checks permissions and existence
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil
}

// buildOpencodeCommand constructs the command to launch opencode.
func (m *Manager) buildOpencodeCommand() string {
	parts := []string{m.config.OpenCode.Command}
	parts = append(parts, m.config.OpenCode.Args...)

	if m.config.OpenCode.Model != "" {
		parts = append(parts, "--model", m.config.OpenCode.Model)
	}
	if m.config.OpenCode.Provider != "" {
		parts = append(parts, "--provider", m.config.OpenCode.Provider)
	}

	return strings.Join(parts, " ")
}

// checkNestedWorktree checks if the current directory is inside a git worktree.
// This prevents creating OCW instances inside worktrees, which would cause issues.
func (m *Manager) checkNestedWorktree() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	current := cwd
	for {
		gitPath := filepath.Join(current, ".git")

		info, err := os.Stat(gitPath)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("cannot create OCW instance inside a git worktree\n\nYou are currently in a worktree at: %s\n\nTo fix:\n  1. Navigate to the main repository root\n  2. Run ocw from there instead", cwd)
			}
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil
}
