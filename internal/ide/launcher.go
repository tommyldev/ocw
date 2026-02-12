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
// 3. Probe for GUI editors (if not headless): cursor, code, zed
// 4. Probe for terminal editors: nvim, vim, vi
// Returns empty string if no editor is found.
func (l *Launcher) DetectEditor() string {
	if l.config.Command != "" {
		if l.commandExists(l.config.Command) {
			return l.config.Command
		}
	}

	if editor := os.Getenv("EDITOR"); editor != "" {
		if l.commandExists(editor) {
			return editor
		}
	}

	isHeadless := l.DetectHeadless()

	if !isHeadless {
		guiEditors := []string{"cursor", "code", "zed"}
		for _, editor := range guiEditors {
			if l.commandExists(editor) {
				return editor
			}
		}
	}

	terminalEditors := []string{"nvim", "vim", "vi"}
	for _, editor := range terminalEditors {
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
		isHeadless := l.DetectHeadless()
		if isHeadless {
			return fmt.Errorf("no editor found\n\nRunning in headless environment (SSH/no display).\n\nTo fix:\n  1. Set EDITOR environment variable: export EDITOR=vim\n  2. Or configure in .ocw/config.toml: editor.command = \"nvim\"\n  3. Install a terminal editor: apt/brew install vim")
		}
		return fmt.Errorf("no editor found\n\nTo fix:\n  1. Set EDITOR environment variable: export EDITOR=code\n  2. Or configure in .ocw/config.toml: editor.command = \"cursor\"\n  3. Install an editor: cursor, vscode, vim, etc.")
	}

	if l.IsTerminalEditor(editor) {
		return l.openTerminalEditor(editor, worktreePath, tmuxTarget)
	}

	if l.DetectHeadless() {
		return fmt.Errorf("GUI editor %q cannot run in headless environment\n\nRunning over SSH or without display server.\n\nTo fix:\n  1. Use a terminal editor instead: export EDITOR=vim\n  2. Or use SSH X11 forwarding: ssh -X\n  3. Or run OCW directly on the machine", editor)
	}

	return l.openGUIEditor(editor, worktreePath)
}

// openGUIEditor launches a GUI editor detached from the current process.
func (l *Launcher) openGUIEditor(editor, worktreePath string) error {
	cmd := exec.Command(editor, worktreePath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch GUI editor %q: %w\n\nTo fix:\n  1. Verify editor is installed: which %s\n  2. Check editor is in PATH: echo $PATH\n  3. Try running manually: %s %s", editor, err, editor, editor, worktreePath)
	}

	return nil
}

// openTerminalEditor launches a terminal editor in a new tmux pane.
func (l *Launcher) openTerminalEditor(editor, worktreePath, tmuxTarget string) error {
	cmd := fmt.Sprintf("%s %s", editor, worktreePath)

	if err := l.tmux.SendKeys(tmuxTarget, cmd); err != nil {
		return fmt.Errorf("failed to launch terminal editor %q in tmux: %w\n\nTo fix:\n  1. Verify editor is installed: which %s\n  2. Check tmux target exists: tmux list-panes -t %s\n  3. Try running editor manually in the pane", editor, err, editor, tmuxTarget)
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
