package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete an instance",
	Long:  "Delete an instance and clean up all associated resources (worktree, tmux window, state)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		force, _ := cmd.Flags().GetBool("force")
		deleteBranch, _ := cmd.Flags().GetBool("delete-branch")

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		repoRoot := cwd
		for {
			if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
				break
			}
			parent := filepath.Dir(repoRoot)
			if parent == repoRoot {
				repoRoot = cwd
				break
			}
			repoRoot = parent
		}

		cfg, err := config.LoadConfig(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr, err := workspace.NewManager(repoRoot, cfg)
		if err != nil {
			return fmt.Errorf("failed to create workspace manager: %w", err)
		}

		instance, err := mgr.GetInstance(instanceID)
		if err != nil {
			instances, listErr := mgr.ListInstances()
			if listErr == nil {
				for _, inst := range instances {
					if inst.Name == instanceID {
						instance = &inst
						instanceID = inst.ID
						break
					}
				}
			}

			if instance == nil {
				return fmt.Errorf("instance %q not found\n\nTo fix:\n  1. List all instances: ocw list\n  2. Use the correct instance ID or name\n  3. Check if the instance was already deleted", instanceID)
			}
		}

		if !force {
			fmt.Printf("⚠ Delete instance '%s' (branch: %s)?\n", instance.Name, instance.Branch)
			fmt.Print("This will remove the worktree and kill all processes. [y/N]: ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		if !deleteBranch && instance.Branch != "" && instance.Branch != instance.BaseBranch {
			fmt.Printf("\nDelete branch '%s' too? [y/N]: ", instance.Branch)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				deleteBranch = true
			}
		}

		err = mgr.DeleteInstance(instanceID, force, deleteBranch)
		if err != nil {
			return fmt.Errorf("failed to delete instance: %w", err)
		}

		fmt.Printf("✓ Instance deleted successfully\n")
		fmt.Printf("  ID:       %s\n", instance.ID)
		fmt.Printf("  Name:     %s\n", instance.Name)
		fmt.Printf("  Branch:   %s\n", instance.Branch)
		if deleteBranch {
			fmt.Printf("  Branch deletion: yes\n")
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	deleteCmd.Flags().Bool("delete-branch", false, "Also delete the branch")
	rootCmd.AddCommand(deleteCmd)
}
