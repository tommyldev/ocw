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

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill all instances and the OCW session",
	Long:  "Terminate all instances, kill the tmux session, and clean up all resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

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

		// Get instance count
		instances, err := mgr.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		// Confirmation prompt (unless --force)
		if !force {
			fmt.Printf("⚠️  WARNING: This will kill ALL instances and the OCW tmux session!\n")
			fmt.Printf("\n")
			if len(instances) > 0 {
				fmt.Printf("Instances to be killed:\n")
				for _, inst := range instances {
					fmt.Printf("  • %s (branch: %s)\n", inst.Name, inst.Branch)
				}
				fmt.Printf("\n")
			}
			fmt.Printf("All worktrees will be removed and processes terminated.\n")
			fmt.Printf("Are you sure? [y/N]: ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Kill cancelled.")
				return nil
			}
		}

		// Delete all instances
		for _, inst := range instances {
			fmt.Printf("Deleting instance %s...\n", inst.Name)
			if err := mgr.DeleteInstance(inst.ID, true, false); err != nil {
				fmt.Printf("  ⚠️  Failed to delete instance %s: %v\n", inst.Name, err)
			} else {
				fmt.Printf("  ✓ Instance %s deleted\n", inst.Name)
			}
		}

		// Kill tmux session
		sessionName := mgr.SessionName()
		if mgr.SessionExists() {
			fmt.Printf("Killing tmux session %s...\n", sessionName)
			if err := mgr.KillSession(); err != nil {
				return fmt.Errorf("failed to kill tmux session: %w", err)
			}
			fmt.Printf("  ✓ Tmux session killed\n")
		}

		fmt.Printf("\n✓ All resources cleaned up successfully\n")

		return nil
	},
}

func init() {
	killCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(killCmd)
}
