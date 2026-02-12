package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var newCmd = &cobra.Command{
	Use:   "new <branch>",
	Short: "Create a new instance",
	Long:  "Create a new instance with a dedicated worktree and tmux window",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]
		baseBranch, _ := cmd.Flags().GetString("base")

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
				// Reached filesystem root without finding .git
				repoRoot = cwd
				break
			}
			repoRoot = parent
		}

		// Load configuration
		cfg, err := config.LoadConfig(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create workspace manager
		mgr, err := workspace.NewManager(repoRoot, cfg)
		if err != nil {
			return fmt.Errorf("failed to create workspace manager: %w", err)
		}

		// Use default base branch if not provided
		if baseBranch == "" {
			baseBranch = cfg.Workspace.BaseBranch
		}

		// Create instance
		opts := workspace.CreateOpts{
			Name:       branchName,
			Branch:     branchName,
			BaseBranch: baseBranch,
		}

		instance, err := mgr.CreateInstance(opts)
		if err != nil {
			return fmt.Errorf("failed to create instance: %w", err)
		}

		// Print success message
		fmt.Printf("✓ Instance created successfully\n")
		fmt.Printf("  ID:       %s\n", instance.ID)
		fmt.Printf("  Branch:   %s\n", instance.Branch)
		fmt.Printf("  Base:     %s\n", instance.BaseBranch)
		fmt.Printf("  Worktree: %s\n", instance.WorktreePath)
		fmt.Printf("  Status:   %s\n", instance.Status)

		return nil
	},
}

func init() {
	newCmd.Flags().StringP("base", "b", "", "Base branch to branch from (default: from config)")
	rootCmd.AddCommand(newCmd)
}
