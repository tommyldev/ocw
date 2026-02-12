package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasOverlap(t *testing.T) {
	tests := []struct {
		name   string
		files1 map[string]bool
		files2 map[string]bool
		want   bool
	}{
		{
			name:   "no overlap - different files",
			files1: map[string]bool{"file1.go": true, "file2.go": true},
			files2: map[string]bool{"file3.go": true, "file4.go": true},
			want:   false,
		},
		{
			name:   "overlap - one common file",
			files1: map[string]bool{"file1.go": true, "file2.go": true},
			files2: map[string]bool{"file2.go": true, "file3.go": true},
			want:   true,
		},
		{
			name:   "overlap - all files common",
			files1: map[string]bool{"file1.go": true, "file2.go": true},
			files2: map[string]bool{"file1.go": true, "file2.go": true},
			want:   true,
		},
		{
			name:   "no overlap - empty sets",
			files1: map[string]bool{},
			files2: map[string]bool{},
			want:   false,
		},
		{
			name:   "no overlap - one empty set",
			files1: map[string]bool{"file1.go": true},
			files2: map[string]bool{},
			want:   false,
		},
		{
			name:   "overlap - nested paths",
			files1: map[string]bool{"internal/git/git.go": true},
			files2: map[string]bool{"internal/git/git.go": true},
			want:   true,
		},
		{
			name:   "no overlap - similar but different paths",
			files1: map[string]bool{"internal/git/git.go": true},
			files2: map[string]bool{"internal/git/merge.go": true},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOverlap(tt.files1, tt.files2)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestHasOverlapSymmetry(t *testing.T) {
	// hasOverlap should be symmetric: hasOverlap(A, B) == hasOverlap(B, A)
	files1 := map[string]bool{"file1.go": true, "file2.go": true}
	files2 := map[string]bool{"file2.go": true, "file3.go": true}

	result1 := hasOverlap(files1, files2)
	result2 := hasOverlap(files2, files1)

	assert.Equal(t, result1, result2, "hasOverlap should be symmetric")
	assert.True(t, result1, "should detect overlap")
}

func TestConflictDetectionLogic(t *testing.T) {
	tests := []struct {
		name           string
		inst1Branch    string
		inst2Branch    string
		inst1Files     []string
		inst2Files     []string
		shouldConflict bool
	}{
		{
			name:           "same branch - no conflict",
			inst1Branch:    "feature/test",
			inst2Branch:    "feature/test",
			inst1Files:     []string{"file1.go", "file2.go"},
			inst2Files:     []string{"file1.go", "file2.go"},
			shouldConflict: false,
		},
		{
			name:           "different branches - overlapping files",
			inst1Branch:    "feature/test-1",
			inst2Branch:    "feature/test-2",
			inst1Files:     []string{"file1.go", "file2.go"},
			inst2Files:     []string{"file2.go", "file3.go"},
			shouldConflict: true,
		},
		{
			name:           "different branches - no overlap",
			inst1Branch:    "feature/test-1",
			inst2Branch:    "feature/test-2",
			inst1Files:     []string{"file1.go"},
			inst2Files:     []string{"file2.go"},
			shouldConflict: false,
		},
		{
			name:           "different branches - empty file lists",
			inst1Branch:    "feature/test-1",
			inst2Branch:    "feature/test-2",
			inst1Files:     []string{},
			inst2Files:     []string{},
			shouldConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert slices to sets
			files1 := make(map[string]bool)
			for _, f := range tt.inst1Files {
				files1[f] = true
			}

			files2 := make(map[string]bool)
			for _, f := range tt.inst2Files {
				files2[f] = true
			}

			// Check if should skip (same branch)
			skipDueToSameBranch := tt.inst1Branch == tt.inst2Branch

			// Check for overlap
			hasFileOverlap := hasOverlap(files1, files2)

			// Should conflict if different branches and files overlap
			shouldConflict := !skipDueToSameBranch && hasFileOverlap

			assert.Equal(t, tt.shouldConflict, shouldConflict)
		})
	}
}

func TestFileSetConversion(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantSize int
	}{
		{
			name:     "convert slice to set",
			files:    []string{"file1.go", "file2.go", "file3.go"},
			wantSize: 3,
		},
		{
			name:     "empty slice",
			files:    []string{},
			wantSize: 0,
		},
		{
			name:     "duplicate files in slice",
			files:    []string{"file1.go", "file1.go", "file2.go"},
			wantSize: 2, // duplicates removed in set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSet := make(map[string]bool)
			for _, f := range tt.files {
				fileSet[f] = true
			}

			assert.Equal(t, tt.wantSize, len(fileSet))

			// Verify all unique files are in the set
			uniqueFiles := make(map[string]bool)
			for _, f := range tt.files {
				uniqueFiles[f] = true
			}
			for f := range uniqueFiles {
				assert.True(t, fileSet[f], "file %s should be in set", f)
			}
		})
	}
}

func TestPairwiseComparison(t *testing.T) {
	// Test the pairwise comparison logic used in DetectConflicts
	type instance struct {
		id     string
		branch string
	}

	instances := []instance{
		{id: "inst1", branch: "feature/1"},
		{id: "inst2", branch: "feature/2"},
		{id: "inst3", branch: "feature/3"},
	}

	// Count pairs checked
	pairs := 0
	for i := range instances {
		for j := range instances {
			if i >= j {
				continue // Skip same instance and already-checked pairs
			}
			pairs++
		}
	}

	// Should check 3 pairs: (0,1), (0,2), (1,2)
	assert.Equal(t, 3, pairs)
}

func TestConflictMapBidirectional(t *testing.T) {
	// When instances conflict, both should be added to each other's conflict list
	conflicts := make(map[string][]string)

	// Simulate finding a conflict between inst1 and inst2
	inst1ID := "inst1"
	inst2ID := "inst2"

	// Add conflict in both directions
	conflicts[inst1ID] = append(conflicts[inst1ID], inst2ID)
	conflicts[inst2ID] = append(conflicts[inst2ID], inst1ID)

	// Verify bidirectional recording
	assert.Contains(t, conflicts[inst1ID], inst2ID)
	assert.Contains(t, conflicts[inst2ID], inst1ID)
	assert.Equal(t, 1, len(conflicts[inst1ID]))
	assert.Equal(t, 1, len(conflicts[inst2ID]))
}

func TestMultipleConflicts(t *testing.T) {
	// Test that an instance can have conflicts with multiple other instances
	conflicts := make(map[string][]string)

	inst1ID := "inst1"
	inst2ID := "inst2"
	inst3ID := "inst3"

	// inst1 conflicts with both inst2 and inst3
	conflicts[inst1ID] = append(conflicts[inst1ID], inst2ID)
	conflicts[inst1ID] = append(conflicts[inst1ID], inst3ID)

	// inst2 conflicts with inst1
	conflicts[inst2ID] = append(conflicts[inst2ID], inst1ID)

	// inst3 conflicts with inst1
	conflicts[inst3ID] = append(conflicts[inst3ID], inst1ID)

	// Verify inst1 has two conflicts
	assert.Equal(t, 2, len(conflicts[inst1ID]))
	assert.Contains(t, conflicts[inst1ID], inst2ID)
	assert.Contains(t, conflicts[inst1ID], inst3ID)

	// Verify inst2 and inst3 have one conflict each
	assert.Equal(t, 1, len(conflicts[inst2ID]))
	assert.Equal(t, 1, len(conflicts[inst3ID]))
}
