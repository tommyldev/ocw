package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/ide"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var editCmd = &cobra.Command{
	Use:   "edit <id|name>",
	Short: "Open an instance's worktree in an editor",
	Long:  "Open an instance's worktree in your configured IDE or $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Find git repository root
		repoRoot := cwd
		for {
			if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
				break
			}
			parent := filepath.Dir(repoRoot)
			if parent == repoRoot {
				return fmt.Errorf("not in a git repository")
			}
			repoRoot = parent
		}

		// Check if .ocw exists
		ocwDir := filepath.Join(repoRoot, ".ocw")
		if _, err := os.Stat(ocwDir); os.IsNotExist(err) {
			return fmt.Errorf(".ocw directory not found; run 'ocw init' first")
		}

		cfg, err := config.LoadConfig(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr, err := workspace.NewManager(repoRoot, cfg)
		if err != nil {
			return fmt.Errorf("failed to create workspace manager: %w", err)
		}

		// Look up instance by ID or name
		instance, err := mgr.GetInstance(idOrName)
		if err != nil {
			// Try to find by name
			instances, listErr := mgr.ListInstances()
			if listErr == nil {
				for _, inst := range instances {
					if inst.Name == idOrName {
						instance = &inst
						break
					}
				}
			}

			if instance == nil {
				return fmt.Errorf("instance %q not found", idOrName)
			}
		}

		// Create IDE launcher
		launcher := ide.NewLauncher(cfg.Editor, mgr.Tmux())

		// Detect editor
		editor := launcher.DetectEditor()
		if editor == "" {
			return fmt.Errorf("no editor found; set EDITOR environment variable or configure editor.command in .ocw/config.toml")
		}

		// Build tmux target for terminal editors
		sessionName := mgr.SessionName()
		tmuxTarget := fmt.Sprintf("%s:%s", sessionName, instance.TmuxWindow)

		// Open editor
		if err := launcher.Open(instance.WorktreePath, tmuxTarget); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		if launcher.IsTerminalEditor(editor) {
			fmt.Printf("✓ Launched %s in tmux pane for instance %s\n", editor, instance.Name)
		} else {
			fmt.Printf("✓ Launched %s for instance %s\n", editor, instance.Name)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
