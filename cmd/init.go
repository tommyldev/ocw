package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize OCW workspace",
	Long:  "Initialize OCW workspace by creating .ocw directory with default config and state files",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current working directory
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
				return fmt.Errorf("not in a git repository\n\nOCW requires a git repository to manage worktrees.\n\nTo fix:\n  1. Initialize a git repository: git init\n  2. Or navigate to an existing git repository\n  3. Then run: ocw init")
			}
			repoRoot = parent
		}

		ocwDir := filepath.Join(repoRoot, ".ocw")
		if _, err := os.Stat(ocwDir); err == nil {
			return fmt.Errorf("OCW workspace already initialized\n\nLocation: %s\n\nTo reconfigure:\n  1. Edit the config: %s\n  2. Or remove and reinitialize: rm -rf %s && ocw init", repoRoot, filepath.Join(ocwDir, "config.toml"), ocwDir)
		}

		// Initialize workspace
		if err := config.InitWorkspace(repoRoot); err != nil {
			return fmt.Errorf("failed to initialize workspace: %w", err)
		}

		fmt.Printf("✓ OCW workspace initialized successfully\n")
		fmt.Printf("  Location: %s\n", repoRoot)
		fmt.Printf("  Config:   %s\n", filepath.Join(ocwDir, "config.toml"))
		fmt.Printf("  State:    %s\n", filepath.Join(ocwDir, "state.json"))
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Review and customize %s\n", filepath.Join(ocwDir, "config.toml"))
		fmt.Printf("  2. Run 'ocw' to start the workspace manager\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
