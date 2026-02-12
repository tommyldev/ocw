package tmux

import (
	"fmt"
	"strings"
)

// WindowInfo contains information about a tmux window.
type WindowInfo struct {
	ID     string
	Name   string
	Active bool
}

// NewWindow creates a new window in the specified session.
// Returns the window ID.
func (t *Tmux) NewWindow(session, name, dir string) (string, error) {
	args := []string{"new-window", "-t", session, "-n", name, "-P", "-F", "#{window_id}"}
	if dir != "" {
		args = append(args, "-c", dir)
	}

	output, err := t.run(args...)
	if err != nil {
		return "", fmt.Errorf("failed to create window %q in session %q: %w", name, session, err)
	}

	return strings.TrimSpace(output), nil
}

// KillWindow closes a tmux window.
func (t *Tmux) KillWindow(target string) error {
	if _, err := t.run("kill-window", "-t", target); err != nil {
		return fmt.Errorf("failed to kill window %q: %w", target, err)
	}
	return nil
}

// ListWindows returns information about all windows in a session.
func (t *Tmux) ListWindows(session string) ([]WindowInfo, error) {
	output, err := t.run("list-windows", "-t", session, "-F", "#{window_id}:#{window_name}:#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("failed to list windows for session %q: %w", session, err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	windows := make([]WindowInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		windows = append(windows, WindowInfo{
			ID:     parts[0],
			Name:   parts[1],
			Active: parts[2] == "1",
		})
	}

	return windows, nil
}

// SelectWindow switches focus to the specified window.
func (t *Tmux) SelectWindow(target string) error {
	if _, err := t.run("select-window", "-t", target); err != nil {
		return fmt.Errorf("failed to select window %q: %w", target, err)
	}
	return nil
}

// SendKeys sends keystrokes to a target window or pane, followed by Enter.
func (t *Tmux) SendKeys(target, keys string) error {
	if _, err := t.run("send-keys", "-t", target, keys, "Enter"); err != nil {
		return fmt.Errorf("failed to send keys to %q: %w", target, err)
	}
	return nil
}

// RunInWindow creates a new window and executes a command in it.
// Returns the window ID.
func (t *Tmux) RunInWindow(session, name, dir, command string) (string, error) {
	windowID, err := t.NewWindow(session, name, dir)
	if err != nil {
		return "", err
	}

	if err := t.SendKeys(windowID, command); err != nil {
		return windowID, fmt.Errorf("created window but failed to run command: %w", err)
	}

	return windowID, nil
}
