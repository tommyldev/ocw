package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// PaneInfo contains information about a tmux pane.
type PaneInfo struct {
	ID      string
	PID     int
	Dead    bool
	Command string
}

// SplitWindow splits a window or pane into two panes.
// split should be "horizontal" (left/right) or "vertical" (top/bottom).
// percentage is the size of the new pane (0-100).
// Returns the new pane ID.
func (t *Tmux) SplitWindow(target, dir, split string, percentage int) (string, error) {
	args := []string{"split-window", "-t", target, "-P", "-F", "#{pane_id}"}

	// Map split direction to tmux flags
	switch split {
	case "horizontal":
		args = append(args, "-h")
	case "vertical":
		args = append(args, "-v")
	default:
		return "", fmt.Errorf("invalid split direction %q: must be 'horizontal' or 'vertical'", split)
	}

	if percentage > 0 && percentage <= 100 {
		args = append(args, "-p", strconv.Itoa(percentage))
	}

	if dir != "" {
		args = append(args, "-c", dir)
	}

	output, err := t.run(args...)
	if err != nil {
		return "", fmt.Errorf("failed to split window at %q: %w", target, err)
	}

	return strings.TrimSpace(output), nil
}

// KillPane closes a tmux pane.
func (t *Tmux) KillPane(target string) error {
	if _, err := t.run("kill-pane", "-t", target); err != nil {
		return fmt.Errorf("failed to kill pane %q: %w", target, err)
	}
	return nil
}

// ListPanes returns information about all panes in a window.
func (t *Tmux) ListPanes(window string) ([]PaneInfo, error) {
	output, err := t.run("list-panes", "-t", window, "-F", "#{pane_id}:#{pane_pid}:#{pane_dead}:#{pane_current_command}")
	if err != nil {
		return nil, fmt.Errorf("failed to list panes for window %q: %w", window, err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	panes := make([]PaneInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 4 {
			continue
		}

		pid, _ := strconv.Atoi(parts[1])
		dead := parts[2] == "1"

		panes = append(panes, PaneInfo{
			ID:      parts[0],
			PID:     pid,
			Dead:    dead,
			Command: parts[3],
		})
	}

	return panes, nil
}

// CapturePaneContent captures the visible content of a pane.
// Returns the text content with escape sequences and trailing whitespace preserved.
func (t *Tmux) CapturePaneContent(target string) (string, error) {
	output, err := t.run("capture-pane", "-p", "-e", "-J", "-t", target)
	if err != nil {
		return "", fmt.Errorf("failed to capture pane %q: %w", target, err)
	}
	return output, nil
}

// CapturePaneScrollback captures the full scrollback history of a pane.
// Uses -S - to start from the beginning of scrollback and -E - for the end.
func (t *Tmux) CapturePaneScrollback(target string) (string, error) {
	output, err := t.run("capture-pane", "-p", "-S", "-", "-E", "-", "-t", target)
	if err != nil {
		return "", fmt.Errorf("failed to capture scrollback for pane %q: %w", target, err)
	}
	return output, nil
}

// SetOption sets a tmux option for the specified target (session, window, or pane).
func (t *Tmux) SetOption(target, option, value string) error {
	if _, err := t.run("set-option", "-t", target, option, value); err != nil {
		return fmt.Errorf("failed to set option %q=%q for %q: %w", option, value, target, err)
	}
	return nil
}

// SetRemainOnExit configures whether a pane should remain visible after the process exits.
// This is useful for detecting when a process has finished.
func (t *Tmux) SetRemainOnExit(target string, on bool) error {
	value := "off"
	if on {
		value = "on"
	}

	if err := t.SetOption(target, "remain-on-exit", value); err != nil {
		return fmt.Errorf("failed to set remain-on-exit for %q: %w", target, err)
	}
	return nil
}
