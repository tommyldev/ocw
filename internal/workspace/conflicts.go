package workspace

import (
	"fmt"

	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
)

// ConflictDetector detects conflicts between instances
type ConflictDetector struct {
	git *git.Git
}

// NewConflictDetector creates a new ConflictDetector
func NewConflictDetector(g *git.Git) *ConflictDetector {
	return &ConflictDetector{git: g}
}

// DetectConflicts checks for file modification conflicts between all active instances
// For each pair of instances on different branches, it compares the modified files
// and records conflicts if they overlap.
// Returns a map of instanceID → []conflicting_instance_IDs
func (cd *ConflictDetector) DetectConflicts(instances []state.Instance) (map[string][]string, error) {
	conflicts := make(map[string][]string)

	// Get modified files for each instance
	modifiedFiles := make(map[string]map[string]bool) // instanceID -> set of modified files
	for _, inst := range instances {
		files, err := cd.getModifiedFiles(inst)
		if err != nil {
			// Log error but continue with other instances
			continue
		}
		fileSet := make(map[string]bool)
		for _, f := range files {
			fileSet[f] = true
		}
		modifiedFiles[inst.ID] = fileSet
	}

	// Check each pair of instances for overlapping modifications
	for i, inst1 := range instances {
		for j, inst2 := range instances {
			if i >= j {
				continue // Skip same instance and already-checked pairs
			}

			// Skip if on same branch
			if inst1.Branch == inst2.Branch {
				continue
			}

			// Check for overlapping modified files
			files1 := modifiedFiles[inst1.ID]
			files2 := modifiedFiles[inst2.ID]

			if hasOverlap(files1, files2) {
				// Record conflict in both directions
				conflicts[inst1.ID] = append(conflicts[inst1.ID], inst2.ID)
				conflicts[inst2.ID] = append(conflicts[inst2.ID], inst1.ID)
			}
		}
	}

	return conflicts, nil
}

// CheckMergeConflicts checks if an instance has merge conflicts with its base branch
// Returns: (hasConflicts, conflictFiles, error)
func (cd *ConflictDetector) CheckMergeConflicts(inst state.Instance) (bool, []string, error) {
	// Use the git package's HasConflicts method
	hasConflicts, conflictFiles, err := cd.git.HasConflicts(inst.Branch, inst.BaseBranch)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check merge conflicts: %w", err)
	}

	return hasConflicts, conflictFiles, nil
}

// getModifiedFiles returns the list of files modified in an instance's branch
// compared to its base branch
func (cd *ConflictDetector) getModifiedFiles(inst state.Instance) ([]string, error) {
	files, err := cd.git.DiffNameOnly(inst.BaseBranch, inst.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files for instance %s: %w", inst.ID, err)
	}
	return files, nil
}

// hasOverlap checks if two file sets have any overlapping files
func hasOverlap(files1, files2 map[string]bool) bool {
	for file := range files1 {
		if files2[file] {
			return true
		}
	}
	return false
}

// UpdateInstanceConflicts updates the ConflictsWith field for all instances
// based on detected conflicts
func (cd *ConflictDetector) UpdateInstanceConflicts(store *state.Store, instances []state.Instance) error {
	conflicts, err := cd.DetectConflicts(instances)
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}

	// Update each instance with its conflicts
	for _, inst := range instances {
		conflictingIDs := conflicts[inst.ID]
		if err := store.UpdateInstance(inst.ID, func(i *state.Instance) {
			i.ConflictsWith = conflictingIDs
		}); err != nil {
			return fmt.Errorf("failed to update instance %s: %w", inst.ID, err)
		}
	}

	return nil
}
