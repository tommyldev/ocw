package ide

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/tmux"
)

// Launcher handles editor detection and launching for both GUI and terminal editors.
type Launcher struct {
	config config.EditorConfig
	tmux   *tmux.Tmux
}

// NewLauncher creates a new IDE launcher with the given editor config and tmux instance.
func NewLauncher(cfg config.EditorConfig, tmuxInstance *tmux.Tmux) *Launcher {
	return &Launcher{
		config: cfg,
		tmux:   tmuxInstance,
	}
}

// DetectEditor returns the editor command to use, following this priority order:
// 1. config.Command (if set)
// 2. $EDITOR environment variable
// 3. Probe for: cursor, code, zed, nvim, vim, vi
// Returns empty string if no editor is found.
func (l *Launcher) DetectEditor() string {
	// 1. Check config.Command
	if l.config.Command != "" {
		if l.commandExists(l.config.Command) {
			return l.config.Command
		}
	}

	// 2. Check $EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		if l.commandExists(editor) {
			return editor
		}
	}

	// 3. Probe for common editors in order
	probeOrder := []string{"cursor", "code", "zed", "nvim", "vim", "vi"}
	for _, editor := range probeOrder {
		if l.commandExists(editor) {
			return editor
		}
	}

	return ""
}

// IsTerminalEditor checks if the given editor is a terminal-based editor.
// It checks against the terminal_editors list from config.
func (l *Launcher) IsTerminalEditor(editor string) bool {
	// Extract just the command name (in case it's a full path)
	cmdName := editor
	if idx := strings.LastIndex(editor, "/"); idx >= 0 {
		cmdName = editor[idx+1:]
	}

	// Check against configured terminal editors
	for _, termEditor := range l.config.TerminalEditors {
		if cmdName == termEditor {
			return true
		}
	}

	return false
}

// Open launches the editor for the given worktree path.
// For GUI editors: launches detached with exec.Command().Start()
// For terminal editors: creates a new tmux pane and runs the editor there
func (l *Launcher) Open(worktreePath, tmuxTarget string) error {
	editor := l.DetectEditor()
	if editor == "" {
		return fmt.Errorf("no editor found; set EDITOR env var or configure editor.command in .ocw/config.toml")
	}

	if l.IsTerminalEditor(editor) {
		// Terminal editor: launch in tmux pane
		return l.openTerminalEditor(editor, worktreePath, tmuxTarget)
	}

	// GUI editor: launch detached
	return l.openGUIEditor(editor, worktreePath)
}

// openGUIEditor launches a GUI editor detached from the current process.
func (l *Launcher) openGUIEditor(editor, worktreePath string) error {
	cmd := exec.Command(editor, worktreePath)
	// Detach from current process
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch GUI editor %q: %w", editor, err)
	}

	// Don't wait for the process; it runs independently
	return nil
}

// openTerminalEditor launches a terminal editor in a new tmux pane.
func (l *Launcher) openTerminalEditor(editor, worktreePath, tmuxTarget string) error {
	// Send the editor command to the tmux pane
	// Format: tmux send-keys -t <target> "<editor> <path>" Enter
	cmd := fmt.Sprintf("%s %s", editor, worktreePath)

	if err := l.tmux.SendKeys(tmuxTarget, cmd); err != nil {
		return fmt.Errorf("failed to launch terminal editor %q in tmux: %w", editor, err)
	}

	return nil
}

// DetectHeadless checks if the system is running in a headless environment.
// Returns true if no display server is available (SSH session, container, etc.)
func (l *Launcher) DetectHeadless() bool {
	// Check for display environment variables
	if os.Getenv("DISPLAY") != "" {
		return false // X11 display available
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return false // Wayland display available
	}

	// Check for SSH session
	if os.Getenv("SSH_TTY") != "" {
		return true // SSH session detected
	}

	// If neither display nor SSH_TTY, assume headless
	return true
}

// commandExists checks if a command is available in the system PATH.
func (l *Launcher) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
