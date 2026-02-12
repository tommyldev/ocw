package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var diffCmd = &cobra.Command{
	Use:   "diff <id|name>",
	Short: "Show git diff statistics for an instance",
	Long:  "Display git diff --stat output for the selected instance vs its base branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]

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

		// Load state
		stateData, err := mgr.Store().Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Find instance by ID or name
		var instance *state.Instance
		for i := range stateData.Instances {
			if stateData.Instances[i].ID == idOrName || stateData.Instances[i].Name == idOrName {
				instance = &stateData.Instances[i]
				break
			}
		}

		if instance == nil {
			return fmt.Errorf("instance not found: %s", idOrName)
		}

		// Create git manager for the worktree
		gitMgr := git.NewGit(instance.WorktreePath)

		// Get diff statistics
		diffStat, err := gitMgr.DiffStatBranch(instance.Branch, instance.BaseBranch)
		if err != nil {
			return fmt.Errorf("failed to get diff statistics: %w", err)
		}

		// Get diff files
		diffFiles, err := gitMgr.DiffFiles(fmt.Sprintf("%s..%s", instance.BaseBranch, instance.Branch))
		if err != nil {
			return fmt.Errorf("failed to get diff files: %w", err)
		}

		// Print header
		fmt.Printf("Diff: %s → %s\n", instance.Branch, instance.BaseBranch)
		fmt.Printf("Summary: %s\n\n", diffStat.Summary)

		// Print file list
		for _, file := range diffFiles {
			icon := getStatusIconCLI(file.Status)
			fmt.Printf("%s %s\n", icon, file.Path)
		}

		return nil
	},
}

// getStatusIconCLI returns the icon for a file status in CLI output
func getStatusIconCLI(status string) string {
	switch status {
	case "M":
		return "[M]"
	case "A":
		return "[+]"
	case "D":
		return "[✕]"
	case "R":
		return "[→]"
	default:
		return "[?]"
	}
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
