package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// MergeResult represents the result of a merge-tree operation
type MergeResult struct {
	Clean         bool
	ConflictFiles []string
}

// MergeBase returns the merge base commit SHA between two branches
func (g *Git) MergeBase(branch1, branch2 string) (string, error) {
	output, err := g.run("merge-base", branch1, branch2)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base: %w", err)
	}
	return output, nil
}

// MergeTree performs a merge-tree operation with explicit merge-base
// Returns a MergeResult indicating if the merge is clean and any conflict files
func (g *Git) MergeTree(base, branch1, branch2 string) (MergeResult, error) {
	// Use --write-tree with explicit --merge-base flag
	cmdArgs := []string{"-C", g.repoPath, "merge-tree", "--write-tree", fmt.Sprintf("--merge-base=%s", base), branch1, branch2}
	cmd := exec.Command("git", cmdArgs...)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// If command succeeds with exit code 0, merge is clean
	if err == nil {
		return MergeResult{
			Clean:         true,
			ConflictFiles: []string{},
		}, nil
	}

	// If command exits with non-zero (typically exit code 1), there are conflicts
	// Parse the output to extract conflict information
	lines := strings.Split(outputStr, "\n")
	var conflictFiles []string

	// The output format with conflicts typically includes:
	// - First line: tree SHA (or empty if severe conflicts)
	// - Subsequent lines: conflict information
	// Look for lines indicating conflicted files
	infoMarkerSeen := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and the tree SHA line
		if line == "" {
			continue
		}

		// Look for the "Auto-merging" or conflict markers
		if strings.HasPrefix(line, "CONFLICT") {
			// Extract file path from conflict message
			// Format: "CONFLICT (content): Merge conflict in <file>"
			if strings.Contains(line, "Merge conflict in ") {
				parts := strings.Split(line, "Merge conflict in ")
				if len(parts) > 1 {
					conflictFiles = append(conflictFiles, strings.TrimSpace(parts[1]))
				}
			}
		} else if strings.HasPrefix(line, "Auto-merging ") {
			infoMarkerSeen = true
		}
	}

	// If we found conflict markers, return them
	if len(conflictFiles) > 0 {
		return MergeResult{
			Clean:         false,
			ConflictFiles: conflictFiles,
		}, nil
	}

	// If command failed but we couldn't parse conflicts, return error
	if !infoMarkerSeen {
		return MergeResult{}, fmt.Errorf("merge-tree command failed: %w: %s", err, outputStr)
	}

	// Assume conflicts exist even if we couldn't parse them
	return MergeResult{
		Clean:         false,
		ConflictFiles: []string{},
	}, nil
}

// HasConflicts checks if merging branch into baseBranch would create conflicts
// Returns: (hasConflicts, conflictFiles, error)
func (g *Git) HasConflicts(branch, baseBranch string) (bool, []string, error) {
	// First find the merge base
	mergeBase, err := g.MergeBase(branch, baseBranch)
	if err != nil {
		return false, nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Then perform merge-tree with explicit merge-base
	result, err := g.MergeTree(mergeBase, branch, baseBranch)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	return !result.Clean, result.ConflictFiles, nil
}
