package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/state"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var dependCmd = &cobra.Command{
	Use:   "depend <instance> <depends-on-instance>",
	Short: "Set a dependency between instances",
	Long:  "Declare that an instance depends on another, blocking merge until the dependency is merged first",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceRef := args[0]
		dependsOnRef := args[1]
		remove, _ := cmd.Flags().GetBool("remove")

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

		instanceID, err := resolveInstanceID(mgr, instanceRef)
		if err != nil {
			return err
		}

		dependsOnID, err := resolveInstanceID(mgr, dependsOnRef)
		if err != nil {
			return err
		}

		if remove {
			if err := mgr.RemoveDependency(instanceID, dependsOnID); err != nil {
				return fmt.Errorf("failed to remove dependency: %w", err)
			}
			fmt.Printf("✓ Removed dependency: %s no longer depends on %s\n", instanceRef, dependsOnRef)
			return nil
		}

		if err := mgr.AddDependency(instanceID, dependsOnID); err != nil {
			return err
		}

		fmt.Printf("✓ Dependency set: %s depends on %s\n", instanceRef, dependsOnRef)
		fmt.Printf("  %s must be merged before %s can be merged\n", dependsOnRef, instanceRef)
		return nil
	},
}

func resolveInstanceID(mgr *workspace.Manager, ref string) (string, error) {
	inst, err := mgr.GetInstance(ref)
	if err == nil {
		return inst.ID, nil
	}

	instances, listErr := mgr.ListInstances()
	if listErr != nil {
		return "", fmt.Errorf("instance %q not found: %w", ref, err)
	}

	for _, inst := range instances {
		if inst.Name == ref || inst.Branch == ref {
			return inst.ID, nil
		}
	}

	return "", fmt.Errorf("instance %q not found\n\nTo fix:\n  1. List all instances: ocw list\n  2. Use the correct instance ID, name, or branch", ref)
}

var dependListCmd = &cobra.Command{
	Use:   "list [instance]",
	Short: "List dependencies for an instance or all instances",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		instances, err := mgr.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		if len(args) == 1 {
			instanceID, err := resolveInstanceID(mgr, args[0])
			if err != nil {
				return err
			}

			var filtered []state.Instance
			for _, inst := range instances {
				if inst.ID == instanceID {
					filtered = append(filtered, inst)
					break
				}
			}
			instances = filtered
		}

		sorted, err := workspace.TopologicalSort(instances)
		if err != nil {
			return fmt.Errorf("dependency error: %w", err)
		}

		allInstances, _ := mgr.ListInstances()

		fmt.Printf("Dependency order (merge from top to bottom):\n\n")
		for i, inst := range sorted {
			depInfo := workspace.FormatDependencyInfo(inst, allInstances)
			if depInfo != "" {
				fmt.Printf("  %d. %s (%s) — %s\n", i+1, inst.Name, inst.ID, depInfo)
			} else {
				fmt.Printf("  %d. %s (%s)\n", i+1, inst.Name, inst.ID)
			}
		}

		return nil
	},
}

func init() {
	dependCmd.Flags().Bool("remove", false, "Remove the dependency instead of adding it")
	dependCmd.AddCommand(dependListCmd)
	rootCmd.AddCommand(dependCmd)
}
