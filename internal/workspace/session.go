package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SessionName returns the tmux session name for this repository.
// Format: <prefix>-<reponame>
// Example: "ocw-myproject"
func (m *Manager) SessionName() string {
	repoName := extractRepoName(m.repoRoot)
	return fmt.Sprintf("%s-%s", m.config.Tmux.SessionPrefix, repoName)
}

// EnsureSession creates the OCW tmux session if it doesn't already exist.
// Returns the session name.
func (m *Manager) EnsureSession() (string, error) {
	sessionName := m.SessionName()

	// Check if session already exists
	if m.tmux.HasSession(sessionName) {
		return sessionName, nil
	}

	// Create new session in the repository root
	if err := m.tmux.NewSession(sessionName, m.repoRoot); err != nil {
		return "", fmt.Errorf("failed to create tmux session %q: %w", sessionName, err)
	}

	return sessionName, nil
}

// SessionExists checks if the OCW tmux session exists.
func (m *Manager) SessionExists() bool {
	return m.tmux.HasSession(m.SessionName())
}

// KillSession terminates the OCW tmux session.
func (m *Manager) KillSession() error {
	sessionName := m.SessionName()
	if !m.tmux.HasSession(sessionName) {
		return fmt.Errorf("session %q does not exist", sessionName)
	}

	if err := m.tmux.KillSession(sessionName); err != nil {
		return fmt.Errorf("failed to kill session %q: %w", sessionName, err)
	}

	return nil
}

// extractRepoName extracts the repository name from a path.
// Example: "/home/user/projects/myrepo" -> "myrepo"
func extractRepoName(repoPath string) string {
	// Get the base name of the path
	name := filepath.Base(repoPath)

	// Clean the name for use in tmux session names
	// Replace any non-alphanumeric characters with hyphens
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)

	// Remove leading/trailing hyphens
	cleaned = strings.Trim(cleaned, "-")

	// If empty after cleaning, use "repo" as fallback
	if cleaned == "" {
		return "repo"
	}

	return cleaned
}
