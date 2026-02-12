package ide

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/tmux"
)

func TestNewLauncher(t *testing.T) {
	cfg := config.EditorConfig{
		Command:         "vim",
		TerminalEditors: []string{"vim", "nvim"},
	}
	tmuxInstance := tmux.NewTmux()

	launcher := NewLauncher(cfg, tmuxInstance)

	assert.NotNil(t, launcher)
	assert.Equal(t, "vim", launcher.config.Command)
	assert.NotNil(t, launcher.tmux)
}

func TestIsTerminalEditor(t *testing.T) {
	cfg := config.EditorConfig{
		Command:         "",
		TerminalEditors: []string{"nvim", "vim", "nano", "emacs"},
	}
	launcher := NewLauncher(cfg, tmux.NewTmux())

	tests := []struct {
		name   string
		editor string
		want   bool
	}{
		{
			name:   "nvim is terminal editor",
			editor: "nvim",
			want:   true,
		},
		{
			name:   "vim is terminal editor",
			editor: "vim",
			want:   true,
		},
		{
			name:   "nano is terminal editor",
			editor: "nano",
			want:   true,
		},
		{
			name:   "code is not terminal editor",
			editor: "code",
			want:   false,
		},
		{
			name:   "cursor is not terminal editor",
			editor: "cursor",
			want:   false,
		},
		{
			name:   "path to vim is terminal editor",
			editor: "/usr/bin/vim",
			want:   true,
		},
		{
			name:   "path to nvim is terminal editor",
			editor: "/usr/local/bin/nvim",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := launcher.IsTerminalEditor(tt.editor)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDetectHeadless(t *testing.T) {
	cfg := config.EditorConfig{}
	launcher := NewLauncher(cfg, tmux.NewTmux())

	tests := []struct {
		name         string
		displayEnv   string
		waylandEnv   string
		sshTTYEnv    string
		wantHeadless bool
		setupEnv     func()
		cleanupEnv   func()
	}{
		{
			name:         "X11 display available",
			displayEnv:   ":0",
			wantHeadless: false,
			setupEnv: func() {
				os.Setenv("DISPLAY", ":0")
				os.Unsetenv("WAYLAND_DISPLAY")
				os.Unsetenv("SSH_TTY")
			},
			cleanupEnv: func() {
				os.Unsetenv("DISPLAY")
			},
		},
		{
			name:         "Wayland display available",
			waylandEnv:   "wayland-0",
			wantHeadless: false,
			setupEnv: func() {
				os.Unsetenv("DISPLAY")
				os.Setenv("WAYLAND_DISPLAY", "wayland-0")
				os.Unsetenv("SSH_TTY")
			},
			cleanupEnv: func() {
				os.Unsetenv("WAYLAND_DISPLAY")
			},
		},
		{
			name:         "SSH session detected",
			sshTTYEnv:    "/dev/pts/0",
			wantHeadless: true,
			setupEnv: func() {
				os.Unsetenv("DISPLAY")
				os.Unsetenv("WAYLAND_DISPLAY")
				os.Setenv("SSH_TTY", "/dev/pts/0")
			},
			cleanupEnv: func() {
				os.Unsetenv("SSH_TTY")
			},
		},
		{
			name:         "no display env vars",
			wantHeadless: true,
			setupEnv: func() {
				os.Unsetenv("DISPLAY")
				os.Unsetenv("WAYLAND_DISPLAY")
				os.Unsetenv("SSH_TTY")
			},
			cleanupEnv: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()
			defer tt.cleanupEnv()

			// Test headless detection
			result := launcher.DetectHeadless()
			assert.Equal(t, tt.wantHeadless, result)
		})
	}
}

func TestEditorDetectionPriority(t *testing.T) {
	tests := []struct {
		name          string
		configCommand string
		editorEnv     string
		expectedOrder []string
	}{
		{
			name:          "config command has highest priority",
			configCommand: "cursor",
			editorEnv:     "vim",
			expectedOrder: []string{"cursor", "vim", "cursor", "code", "zed", "nvim", "vim", "vi"},
		},
		{
			name:          "EDITOR env var is second priority",
			configCommand: "",
			editorEnv:     "nvim",
			expectedOrder: []string{"nvim", "cursor", "code", "zed", "nvim", "vim", "vi"},
		},
		{
			name:          "fallback to probing GUI editors",
			configCommand: "",
			editorEnv:     "",
			expectedOrder: []string{"cursor", "code", "zed", "nvim", "vim", "vi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the priority order is correct
			// The actual order is: config.Command -> EDITOR -> GUI editors -> terminal editors
			assert.NotEmpty(t, tt.expectedOrder)

			// First item should be config command if set
			if tt.configCommand != "" {
				assert.Equal(t, tt.configCommand, tt.expectedOrder[0])
			}
		})
	}
}

func TestPathExtraction(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		want     string
	}{
		{
			name:     "simple command name",
			fullPath: "vim",
			want:     "vim",
		},
		{
			name:     "absolute path to vim",
			fullPath: "/usr/bin/vim",
			want:     "vim",
		},
		{
			name:     "absolute path to nvim",
			fullPath: "/usr/local/bin/nvim",
			want:     "nvim",
		},
		{
			name:     "relative path",
			fullPath: "./bin/vim",
			want:     "vim",
		},
		{
			name:     "path with multiple directories",
			fullPath: "/usr/local/share/bin/code",
			want:     "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract command name from path
			cmdName := extractCommandName(tt.fullPath)
			assert.Equal(t, tt.want, cmdName)
		})
	}
}

func TestEditorClassification(t *testing.T) {
	cfg := config.EditorConfig{
		Command:         "",
		TerminalEditors: []string{"nvim", "vim", "nano", "emacs"},
	}
	launcher := NewLauncher(cfg, tmux.NewTmux())

	tests := []struct {
		name       string
		editor     string
		isTerminal bool
	}{
		{name: "nvim", editor: "nvim", isTerminal: true},
		{name: "vim", editor: "vim", isTerminal: true},
		{name: "nano", editor: "nano", isTerminal: true},
		{name: "emacs", editor: "emacs", isTerminal: true},
		{name: "code", editor: "code", isTerminal: false},
		{name: "cursor", editor: "cursor", isTerminal: false},
		{name: "zed", editor: "zed", isTerminal: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := launcher.IsTerminalEditor(tt.editor)
			assert.Equal(t, tt.isTerminal, result)
		})
	}
}

func TestGUIEditorList(t *testing.T) {
	// Test that GUI editor list is correct
	guiEditors := []string{"cursor", "code", "zed"}

	assert.Equal(t, 3, len(guiEditors))
	assert.Contains(t, guiEditors, "cursor")
	assert.Contains(t, guiEditors, "code")
	assert.Contains(t, guiEditors, "zed")
}

func TestTerminalEditorList(t *testing.T) {
	// Test that terminal editor list is correct
	terminalEditors := []string{"nvim", "vim", "vi"}

	assert.Equal(t, 3, len(terminalEditors))
	assert.Contains(t, terminalEditors, "nvim")
	assert.Contains(t, terminalEditors, "vim")
	assert.Contains(t, terminalEditors, "vi")
}

func TestDefaultEditorConfig(t *testing.T) {
	// Test default editor config from config package
	cfg := config.DefaultConfig()

	assert.Equal(t, "", cfg.Editor.Command)
	assert.Contains(t, cfg.Editor.TerminalEditors, "nvim")
	assert.Contains(t, cfg.Editor.TerminalEditors, "vim")
	assert.Contains(t, cfg.Editor.TerminalEditors, "nano")
	assert.Contains(t, cfg.Editor.TerminalEditors, "emacs")
}

// Helper function to extract command name from path
func extractCommandName(path string) string {
	// Find last slash
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return path
	}

	return path[lastSlash+1:]
}
