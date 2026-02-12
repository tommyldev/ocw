package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Tmux provides a stateless abstraction over tmux CLI commands.
type Tmux struct{}

// NewTmux creates a new Tmux command runner.
func NewTmux() *Tmux {
	return &Tmux{}
}

// IsInstalled checks if tmux is available on the system.
func (t *Tmux) IsInstalled() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// Version returns the tmux version string.
func (t *Tmux) Version() (string, error) {
	output, err := t.run("-V")
	if err != nil {
		return "", fmt.Errorf("failed to get tmux version: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// run executes a tmux command and captures its output.
// Returns stdout as a string or an error if the command fails.
func (t *Tmux) run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("tmux command failed: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("tmux command failed: %w", err)
	}

	return stdout.String(), nil
}

// runAttached executes a tmux command with stdin/stdout/stderr attached to the current terminal.
// This is used for interactive commands like attach-session.
func (t *Tmux) runAttached(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux command failed: %w", err)
	}

	return nil
}
