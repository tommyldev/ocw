package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the complete OCW configuration
type Config struct {
	Workspace WorkspaceConfig `toml:"workspace"`
	OpenCode  OpenCodeConfig  `toml:"opencode"`
	Editor    EditorConfig    `toml:"editor"`
	Merge     MergeConfig     `toml:"merge"`
	Tmux      TmuxConfig      `toml:"tmux"`
	UI        UIConfig        `toml:"ui"`
}

// WorkspaceConfig contains worktree-related settings
type WorkspaceConfig struct {
	WorktreeDir string `toml:"worktree_dir"`
	BaseBranch  string `toml:"base_branch"`
}

// OpenCodeConfig contains OpenCode CLI settings
type OpenCodeConfig struct {
	Command  string   `toml:"command"`
	Args     []string `toml:"args"`
	Model    string   `toml:"model"`
	Provider string   `toml:"provider"`
}

// EditorConfig contains editor settings
type EditorConfig struct {
	Command         string   `toml:"command"`
	TerminalEditors []string `toml:"terminal_editors"`
}

// MergeConfig contains PR and merge settings
type MergeConfig struct {
	Provider           string `toml:"provider"`
	AutoDeleteBranch   bool   `toml:"auto_delete_branch"`
	AutoDeleteWorktree bool   `toml:"auto_delete_worktree"`
	DraftPR            bool   `toml:"draft_pr"`
	PRTemplate         string `toml:"pr_template"`
}

// TmuxConfig contains tmux session settings
type TmuxConfig struct {
	SessionPrefix    string `toml:"session_prefix"`
	AttachOnCreate   bool   `toml:"attach_on_create"`
	DefaultSplit     string `toml:"default_split"`
	PrimaryPaneRatio int    `toml:"primary_pane_ratio"`
}

// UIConfig contains UI display settings
type UIConfig struct {
	ShowElapsedTime      bool `toml:"show_elapsed_time"`
	ShowLastOutput       bool `toml:"show_last_output"`
	ShowSubTerminalCount bool `toml:"show_sub_terminal_count"`
	ShowConflictWarnings bool `toml:"show_conflict_warnings"`
	MaxInstances         int  `toml:"max_instances"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Workspace: WorkspaceConfig{
			WorktreeDir: ".worktrees",
			BaseBranch:  "master",
		},
		OpenCode: OpenCodeConfig{
			Command:  "opencode",
			Args:     []string{},
			Model:    "claude-sonnet-4-5",
			Provider: "Sisyphus",
		},
		Editor: EditorConfig{
			Command:         "",
			TerminalEditors: []string{"nvim", "vim", "nano", "emacs"},
		},
		Merge: MergeConfig{
			Provider:           "github",
			AutoDeleteBranch:   false,
			AutoDeleteWorktree: false,
			DraftPR:            false,
			PRTemplate:         "",
		},
		Tmux: TmuxConfig{
			SessionPrefix:    "ocw",
			AttachOnCreate:   false,
			DefaultSplit:     "horizontal",
			PrimaryPaneRatio: 70,
		},
		UI: UIConfig{
			ShowElapsedTime:      true,
			ShowLastOutput:       true,
			ShowSubTerminalCount: true,
			ShowConflictWarnings: true,
			MaxInstances:         10,
		},
	}
}

// LoadConfig reads the config.toml file from the specified directory.
// Missing fields are filled with defaults from DefaultConfig().
func LoadConfig(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ".ocw", "config.toml")

	// Start with defaults
	cfg := DefaultConfig()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return defaults if file doesn't exist
		return cfg, nil
	}

	// Read and decode TOML
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes the config to config.toml in the specified directory
func SaveConfig(dir string, cfg *Config) error {
	configPath := filepath.Join(dir, ".ocw", "config.toml")

	// Ensure .ocw directory exists
	ocwDir := filepath.Join(dir, ".ocw")
	if err := os.MkdirAll(ocwDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ocw directory: %w", err)
	}

	// Create config file
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	// Encode to TOML
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
