package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)

	// Workspace defaults
	assert.Equal(t, ".worktrees", cfg.Workspace.WorktreeDir)
	assert.Equal(t, "master", cfg.Workspace.BaseBranch)

	// OpenCode defaults
	assert.Equal(t, "opencode", cfg.OpenCode.Command)
	assert.Equal(t, []string{}, cfg.OpenCode.Args)
	assert.Equal(t, "claude-sonnet-4-5", cfg.OpenCode.Model)
	assert.Equal(t, "Sisyphus", cfg.OpenCode.Provider)

	// Editor defaults
	assert.Equal(t, "", cfg.Editor.Command)
	assert.Contains(t, cfg.Editor.TerminalEditors, "nvim")
	assert.Contains(t, cfg.Editor.TerminalEditors, "vim")

	// Merge defaults
	assert.Equal(t, "github", cfg.Merge.Provider)
	assert.Equal(t, false, cfg.Merge.AutoDeleteBranch)
	assert.Equal(t, false, cfg.Merge.AutoDeleteWorktree)
	assert.Equal(t, false, cfg.Merge.DraftPR)

	// Tmux defaults
	assert.Equal(t, "ocw", cfg.Tmux.SessionPrefix)
	assert.Equal(t, false, cfg.Tmux.AttachOnCreate)
	assert.Equal(t, "horizontal", cfg.Tmux.DefaultSplit)
	assert.Equal(t, 70, cfg.Tmux.PrimaryPaneRatio)

	// UI defaults
	assert.Equal(t, true, cfg.UI.ShowElapsedTime)
	assert.Equal(t, true, cfg.UI.ShowLastOutput)
	assert.Equal(t, true, cfg.UI.ShowSubTerminalCount)
	assert.Equal(t, true, cfg.UI.ShowConflictWarnings)
	assert.Equal(t, 10, cfg.UI.MaxInstances)
}

func TestConfigTOMLRoundtrip(t *testing.T) {
	cfg := DefaultConfig()

	// Modify some values
	cfg.Workspace.WorktreeDir = "/custom/worktrees"
	cfg.Workspace.BaseBranch = "main"
	cfg.OpenCode.Model = "custom-model"
	cfg.Tmux.SessionPrefix = "custom-ocw"

	// Encode to TOML
	tmpFile := filepath.Join(t.TempDir(), "config.toml")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	encoder := toml.NewEncoder(f)
	err = encoder.Encode(cfg)
	require.NoError(t, err)
	f.Close()

	// Decode back
	var decoded Config
	_, err = toml.DecodeFile(tmpFile, &decoded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, cfg.Workspace.WorktreeDir, decoded.Workspace.WorktreeDir)
	assert.Equal(t, cfg.Workspace.BaseBranch, decoded.Workspace.BaseBranch)
	assert.Equal(t, cfg.OpenCode.Model, decoded.OpenCode.Model)
	assert.Equal(t, cfg.Tmux.SessionPrefix, decoded.Tmux.SessionPrefix)
}

func TestLoadConfigNonexistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Load config from directory without config file
	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)

	// Should return default config
	assert.NotNil(t, cfg)
	assert.Equal(t, ".worktrees", cfg.Workspace.WorktreeDir)
	assert.Equal(t, "master", cfg.Workspace.BaseBranch)
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom config
	cfg := DefaultConfig()
	cfg.Workspace.WorktreeDir = "/test/worktrees"
	cfg.Workspace.BaseBranch = "main"
	cfg.OpenCode.Model = "test-model"
	cfg.Tmux.SessionPrefix = "test-ocw"
	cfg.UI.MaxInstances = 20

	// Save config
	err := SaveConfig(tmpDir, cfg)
	require.NoError(t, err)

	// Verify .ocw directory was created
	ocwDir := filepath.Join(tmpDir, ".ocw")
	_, err = os.Stat(ocwDir)
	assert.NoError(t, err)

	// Verify config.toml was created
	configPath := filepath.Join(ocwDir, "config.toml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load config
	loaded, err := LoadConfig(tmpDir)
	require.NoError(t, err)

	// Verify loaded config matches saved config
	assert.Equal(t, cfg.Workspace.WorktreeDir, loaded.Workspace.WorktreeDir)
	assert.Equal(t, cfg.Workspace.BaseBranch, loaded.Workspace.BaseBranch)
	assert.Equal(t, cfg.OpenCode.Model, loaded.OpenCode.Model)
	assert.Equal(t, cfg.Tmux.SessionPrefix, loaded.Tmux.SessionPrefix)
	assert.Equal(t, cfg.UI.MaxInstances, loaded.UI.MaxInstances)
}

func TestLoadConfigPartialOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ocw directory
	ocwDir := filepath.Join(tmpDir, ".ocw")
	err := os.MkdirAll(ocwDir, 0755)
	require.NoError(t, err)

	// Write partial config (only override some values)
	configPath := filepath.Join(ocwDir, "config.toml")
	partialConfig := `
[workspace]
worktree_dir = "/custom/worktrees"
base_branch = "develop"

[ui]
max_instances = 25
`
	err = os.WriteFile(configPath, []byte(partialConfig), 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)

	// Overridden values should be set
	assert.Equal(t, "/custom/worktrees", cfg.Workspace.WorktreeDir)
	assert.Equal(t, "develop", cfg.Workspace.BaseBranch)
	assert.Equal(t, 25, cfg.UI.MaxInstances)

	// Default values should still be present
	assert.Equal(t, "opencode", cfg.OpenCode.Command)
	assert.Equal(t, "github", cfg.Merge.Provider)
	assert.Equal(t, "ocw", cfg.Tmux.SessionPrefix)
}

func TestConfigStructFields(t *testing.T) {
	cfg := Config{
		Workspace: WorkspaceConfig{
			WorktreeDir: "/test",
			BaseBranch:  "main",
		},
		OpenCode: OpenCodeConfig{
			Command:  "custom-opencode",
			Args:     []string{"--verbose"},
			Model:    "custom-model",
			Provider: "custom-provider",
		},
		Editor: EditorConfig{
			Command:         "vim",
			TerminalEditors: []string{"vim", "nvim"},
		},
		Merge: MergeConfig{
			Provider:           "gitlab",
			AutoDeleteBranch:   true,
			AutoDeleteWorktree: true,
			DraftPR:            true,
			PRTemplate:         "template.md",
		},
		Tmux: TmuxConfig{
			SessionPrefix:    "test",
			AttachOnCreate:   true,
			DefaultSplit:     "vertical",
			PrimaryPaneRatio: 60,
		},
		UI: UIConfig{
			ShowElapsedTime:      false,
			ShowLastOutput:       false,
			ShowSubTerminalCount: false,
			ShowConflictWarnings: false,
			MaxInstances:         5,
		},
	}

	// Verify all fields are accessible
	assert.Equal(t, "/test", cfg.Workspace.WorktreeDir)
	assert.Equal(t, "main", cfg.Workspace.BaseBranch)
	assert.Equal(t, "custom-opencode", cfg.OpenCode.Command)
	assert.Equal(t, []string{"--verbose"}, cfg.OpenCode.Args)
	assert.Equal(t, "vim", cfg.Editor.Command)
	assert.Equal(t, "gitlab", cfg.Merge.Provider)
	assert.Equal(t, true, cfg.Merge.AutoDeleteBranch)
	assert.Equal(t, "test", cfg.Tmux.SessionPrefix)
	assert.Equal(t, true, cfg.Tmux.AttachOnCreate)
	assert.Equal(t, false, cfg.UI.ShowElapsedTime)
	assert.Equal(t, 5, cfg.UI.MaxInstances)
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure .ocw doesn't exist yet
	ocwDir := filepath.Join(tmpDir, ".ocw")
	_, err := os.Stat(ocwDir)
	assert.True(t, os.IsNotExist(err))

	// Save config
	cfg := DefaultConfig()
	err = SaveConfig(tmpDir, cfg)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(ocwDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestConfigTOMLTags(t *testing.T) {
	// Test that TOML marshaling uses correct field names
	cfg := DefaultConfig()
	cfg.Workspace.WorktreeDir = "/test/dir"

	tmpFile := filepath.Join(t.TempDir(), "config.toml")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)

	encoder := toml.NewEncoder(f)
	err = encoder.Encode(cfg)
	require.NoError(t, err)
	f.Close()

	// Read the file and check for correct field names
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	content := string(data)
	// Should use snake_case field names
	assert.Contains(t, content, "worktree_dir")
	assert.Contains(t, content, "base_branch")
	assert.Contains(t, content, "session_prefix")
	assert.Contains(t, content, "max_instances")
}
