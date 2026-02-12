package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var focusCmd = &cobra.Command{
	Use:   "focus <id|name>",
	Short: "Attach to an instance's tmux window",
	Long:  "Attach directly to an instance's tmux window, taking over the terminal",
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

		// Get session name
		sessionName := mgr.SessionName()

		// Attach to the specific window
		if err := mgr.Tmux().AttachWindow(sessionName, instance.TmuxWindow); err != nil {
			return fmt.Errorf("failed to attach to instance window: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(focusCmd)
}
