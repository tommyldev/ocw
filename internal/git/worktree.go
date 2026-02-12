package git

import (
	"fmt"
	"strings"
)

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path     string
	Branch   string
	Head     string
	Bare     bool
	Detached bool
}

// WorktreeAdd creates a new worktree at the specified path with a new branch based on base
func (g *Git) WorktreeAdd(path, branch, base string) error {
	_, err := g.run("worktree", "add", path, "-b", branch, base)
	if err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}
	return nil
}

// WorktreeAddExisting creates a new worktree at the specified path for an existing branch
func (g *Git) WorktreeAddExisting(path, branch string) error {
	_, err := g.run("worktree", "add", path, branch)
	if err != nil {
		return fmt.Errorf("failed to add worktree for existing branch: %w", err)
	}
	return nil
}

// WorktreeRemove removes the worktree at the specified path
func (g *Git) WorktreeRemove(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, path)

	_, err := g.run(args...)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

// WorktreeList returns a list of all worktrees by parsing `git worktree list --porcelain`
func (g *Git) WorktreeList() ([]WorktreeInfo, error) {
	output, err := g.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeListPorcelain(output), nil
}

// parseWorktreeListPorcelain parses the porcelain format output from git worktree list
// Format:
// worktree /path/to/worktree
// HEAD abc123...
// branch refs/heads/branch-name
// (blank line)
// worktree /path/to/another
// HEAD def456...
// detached
func parseWorktreeListPorcelain(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo

	if output == "" {
		return worktrees
	}

	lines := strings.Split(output, "\n")
	var current WorktreeInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Empty line indicates end of worktree entry
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		// Parse worktree fields
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			branchRef := strings.TrimPrefix(line, "branch ")
			// Extract branch name from refs/heads/branch-name
			if strings.HasPrefix(branchRef, "refs/heads/") {
				current.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
			} else {
				current.Branch = branchRef
			}
		} else if line == "bare" {
			current.Bare = true
		} else if line == "detached" {
			current.Detached = true
		}
	}

	// Add the last worktree if there's no trailing newline
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// WorktreeRepair repairs worktree administrative files
func (g *Git) WorktreeRepair() error {
	_, err := g.run("worktree", "repair")
	if err != nil {
		return fmt.Errorf("failed to repair worktrees: %w", err)
	}
	return nil
}

// WorktreePrune removes worktree information for deleted worktrees
func (g *Git) WorktreePrune() error {
	_, err := g.run("worktree", "prune")
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}
	return nil
}
