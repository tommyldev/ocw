package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeResultStruct(t *testing.T) {
	tests := []struct {
		name          string
		clean         bool
		conflictFiles []string
	}{
		{
			name:          "clean merge",
			clean:         true,
			conflictFiles: []string{},
		},
		{
			name:          "merge with conflicts",
			clean:         false,
			conflictFiles: []string{"file1.go", "file2.go"},
		},
		{
			name:          "merge with single conflict",
			clean:         false,
			conflictFiles: []string{"README.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeResult{
				Clean:         tt.clean,
				ConflictFiles: tt.conflictFiles,
			}

			assert.Equal(t, tt.clean, result.Clean)
			assert.Equal(t, tt.conflictFiles, result.ConflictFiles)
			assert.Equal(t, len(tt.conflictFiles), len(result.ConflictFiles))
		})
	}
}

func TestMergeTreeOutputParsing(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		wantClean     bool
		wantConflicts []string
	}{
		{
			name:          "clean merge - empty output",
			output:        "",
			wantClean:     true,
			wantConflicts: []string{},
		},
		{
			name:          "single conflict",
			output:        "CONFLICT (content): Merge conflict in file.go",
			wantClean:     false,
			wantConflicts: []string{"file.go"},
		},
		{
			name: "multiple conflicts",
			output: `CONFLICT (content): Merge conflict in file1.go
CONFLICT (content): Merge conflict in file2.go
CONFLICT (content): Merge conflict in README.md`,
			wantClean:     false,
			wantConflicts: []string{"file1.go", "file2.go", "README.md"},
		},
		{
			name: "conflict with auto-merge info",
			output: `Auto-merging file1.go
CONFLICT (content): Merge conflict in file2.go
Auto-merging file3.go`,
			wantClean:     false,
			wantConflicts: []string{"file2.go"},
		},
		{
			name:          "conflict with path containing spaces",
			output:        `CONFLICT (content): Merge conflict in path/to/file name.go`,
			wantClean:     false,
			wantConflicts: []string{"path/to/file name.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic from MergeTree
			lines := parseLines(tt.output)
			var conflictFiles []string

			for _, line := range lines {
				line = trimSpace(line)
				if line == "" {
					continue
				}

				if hasPrefix(line, "CONFLICT") && contains(line, "Merge conflict in ") {
					parts := splitString(line, "Merge conflict in ")
					if len(parts) > 1 {
						conflictFiles = append(conflictFiles, trimSpace(parts[1]))
					}
				}
			}

			clean := len(conflictFiles) == 0
			assert.Equal(t, tt.wantClean, clean)
			if !tt.wantClean {
				assert.Equal(t, tt.wantConflicts, conflictFiles)
			}
		})
	}
}

func TestConflictDetectionPatterns(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		hasConflict  bool
		expectedFile string
	}{
		{
			name:         "standard conflict message",
			line:         "CONFLICT (content): Merge conflict in main.go",
			hasConflict:  true,
			expectedFile: "main.go",
		},
		{
			name:         "conflict with nested path",
			line:         "CONFLICT (content): Merge conflict in internal/git/merge.go",
			hasConflict:  true,
			expectedFile: "internal/git/merge.go",
		},
		{
			name:        "auto-merge without conflict",
			line:        "Auto-merging file.go",
			hasConflict: false,
		},
		{
			name:        "empty line",
			line:        "",
			hasConflict: false,
		},
		{
			name:        "tree SHA",
			line:        "abc123def456",
			hasConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasConflict := hasPrefix(tt.line, "CONFLICT") && contains(tt.line, "Merge conflict in ")
			assert.Equal(t, tt.hasConflict, hasConflict)

			if hasConflict {
				parts := splitString(tt.line, "Merge conflict in ")
				if len(parts) > 1 {
					file := trimSpace(parts[1])
					assert.Equal(t, tt.expectedFile, file)
				}
			}
		})
	}
}

// Helper functions for testing parsing logic

func parseLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := []string{}
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	if sep == "" {
		return []string{s}
	}

	parts := []string{}
	start := 0

	for i := 0; i <= len(s)-len(sep); i++ {
		match := true
		for j := 0; j < len(sep); j++ {
			if s[i+j] != sep[j] {
				match = false
				break
			}
		}
		if match {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}

	parts = append(parts, s[start:])
	return parts
}
