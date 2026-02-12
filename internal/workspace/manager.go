package workspace

import (
	"fmt"

	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/tmux"
)

// Manager orchestrates git worktrees, tmux sessions/windows, and state persistence.
// It provides the core coordination layer for OCW instance lifecycle.
type Manager struct {
	git      *git.Git
	tmux     *tmux.Tmux
	store    *state.Store
	config   *config.Config
	repoRoot string
}

// NewManager creates a new workspace manager.
// repoRoot should be the absolute path to the git repository root.
func NewManager(repoRoot string, cfg *config.Config) (*Manager, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("repoRoot cannot be empty")
	}

	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Initialize git
	gitClient := git.NewGit(repoRoot)
	if !gitClient.IsGitRepo() {
		return nil, fmt.Errorf("path %q is not a git repository", repoRoot)
	}

	// Initialize tmux
	tmuxClient := tmux.NewTmux()
	if !tmuxClient.IsInstalled() {
		return nil, fmt.Errorf("tmux is not installed")
	}

	// Initialize state store
	stateStore := state.NewStore(repoRoot)

	return &Manager{
		git:      gitClient,
		tmux:     tmuxClient,
		store:    stateStore,
		config:   cfg,
		repoRoot: repoRoot,
	}, nil
}

// Git returns the underlying git client
func (m *Manager) Git() *git.Git {
	return m.git
}

// Tmux returns the underlying tmux client
func (m *Manager) Tmux() *tmux.Tmux {
	return m.tmux
}

// Store returns the underlying state store
func (m *Manager) Store() *state.Store {
	return m.store
}

// Config returns the configuration
func (m *Manager) Config() *config.Config {
	return m.config
}

// RepoRoot returns the repository root path
func (m *Manager) RepoRoot() string {
	return m.repoRoot
}
