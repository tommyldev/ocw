package tmux

import (
	"fmt"
	"strings"
)

// NewSession creates a new detached tmux session.
// If dir is empty, uses the current working directory.
func (t *Tmux) NewSession(name, dir string) error {
	args := []string{"new-session", "-d", "-s", name}
	if dir != "" {
		args = append(args, "-c", dir)
	}

	if _, err := t.run(args...); err != nil {
		return fmt.Errorf("failed to create session %q: %w", name, err)
	}

	return nil
}

// HasSession checks if a tmux session with the given name exists.
func (t *Tmux) HasSession(name string) bool {
	_, err := t.run("has-session", "-t", name)
	return err == nil
}

// KillSession terminates a tmux session.
func (t *Tmux) KillSession(name string) error {
	if _, err := t.run("kill-session", "-t", name); err != nil {
		return fmt.Errorf("failed to kill session %q: %w", name, err)
	}
	return nil
}

// ListSessions returns a list of all active tmux session names.
func (t *Tmux) ListSessions() ([]string, error) {
	output, err := t.run("list-sessions", "-F", "#{session_name}")
	if err != nil {
		// If no sessions exist, tmux returns an error
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "failed to connect") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}

	return lines, nil
}

// AttachSession attaches to an existing tmux session, taking over the terminal.
// This is an interactive operation that blocks until the user detaches.
func (t *Tmux) AttachSession(name string) error {
	if err := t.runAttached("attach-session", "-t", name); err != nil {
		return fmt.Errorf("failed to attach to session %q: %w", name, err)
	}
	return nil
}

// AttachWindow attaches to a specific window in a session, taking over the terminal.
// This is an interactive operation that blocks until the user detaches.
// The window is selected before attaching.
func (t *Tmux) AttachWindow(session, window string) error {
	// Build the target: session:window
	target := session + ":" + window
	if err := t.runAttached("attach-session", "-t", target); err != nil {
		return fmt.Errorf("failed to attach to window %q in session %q: %w", window, session, err)
	}
	return nil
}
