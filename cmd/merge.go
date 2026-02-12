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

var mergeCmd = &cobra.Command{
	Use:   "merge <id|name>",
	Short: "Push branch and create a pull request",
	Long:  "Push the instance's branch to origin and create a pull request via gh or glab CLI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]

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

		stateData, err := mgr.Store().Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		var instance *state.Instance
		for i := range stateData.Instances {
			if stateData.Instances[i].ID == idOrName || stateData.Instances[i].Name == idOrName {
				instance = &stateData.Instances[i]
				break
			}
		}

		if instance == nil {
			return fmt.Errorf("instance %q not found\n\nTo fix:\n  1. List all instances: ocw list\n  2. Use the correct instance ID or name", idOrName)
		}

		gitMgr := git.NewGit(instance.WorktreePath)
		conflictDetector := workspace.NewConflictDetector(gitMgr)

		fmt.Printf("Checking for conflicts...\n")
		hasConflicts, conflictFiles, err := conflictDetector.CheckMergeConflicts(*instance)
		if err != nil {
			return fmt.Errorf("failed to check merge conflicts: %w\n\nTo fix:\n  1. Ensure base branch exists: git branch -a | grep %s\n  2. Fetch latest changes: git fetch\n  3. Verify git repository is valid", err, instance.BaseBranch)
		}

		if hasConflicts {
			fmt.Printf("❌ Cannot merge: conflicts detected\n\n")
			fmt.Printf("Conflicting files:\n")
			for _, file := range conflictFiles {
				fmt.Printf("  • %s\n", file)
			}
			return fmt.Errorf("resolve conflicts before merging\n\nTo fix:\n  1. Rebase onto %s: git rebase %s\n  2. Or merge manually: git merge %s\n  3. Resolve conflicts and commit changes", instance.BaseBranch, instance.BaseBranch, instance.BaseBranch)
		}

		fmt.Printf("✓ No conflicts detected\n\n")

		tool, err := mgr.DetectPRTool()
		if err != nil {
			return err
		}

		fmt.Printf("Pushing branch %s to origin...\n", instance.Branch)
		if err := mgr.PushBranch(instance.ID); err != nil {
			return err
		}

		fmt.Printf("✓ Branch pushed\n\n")

		prTitle := formatBranchNameForPR(instance.Branch)
		fmt.Printf("Creating PR with %s...\n", tool)
		fmt.Printf("Title: %s\n", prTitle)

		prURL, err := mgr.CreatePR(instance.ID, prTitle, "")
		if err != nil {
			return err
		}

		fmt.Printf("\n✓ Pull request created successfully!\n")
		fmt.Printf("PR URL: %s\n", prURL)

		return nil
	},
}

func formatBranchNameForPR(branch string) string {
	branch = trimCommonPrefix(branch, "feature/")
	branch = trimCommonPrefix(branch, "feat/")
	branch = trimCommonPrefix(branch, "bugfix/")
	branch = trimCommonPrefix(branch, "fix/")
	branch = trimCommonPrefix(branch, "hotfix/")

	branch = replaceAll(branch, "-", " ")
	branch = replaceAll(branch, "_", " ")

	if len(branch) > 0 {
		branch = toUpper(branch[:1]) + branch[1:]
	}

	return branch
}

func trimCommonPrefix(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

func toUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	b := s[0]
	if b >= 'a' && b <= 'z' {
		return string(b-32) + s[1:]
	}
	return s
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}
