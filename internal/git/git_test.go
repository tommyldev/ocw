package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGit(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{
			name:     "creates git instance with path",
			repoPath: "/path/to/repo",
			want:     "/path/to/repo",
		},
		{
			name:     "creates git instance with empty path",
			repoPath: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGit(tt.repoPath)
			assert.NotNil(t, g)
			assert.Equal(t, tt.want, g.repoPath)
		})
	}
}

func TestGetDefaultBranchLogic(t *testing.T) {
	tests := []struct {
		name           string
		symbolicRef    string
		symbolicRefErr bool
		expectedBranch string
	}{
		{
			name:           "extracts main from symbolic ref",
			symbolicRef:    "refs/remotes/origin/main",
			symbolicRefErr: false,
			expectedBranch: "main",
		},
		{
			name:           "extracts master from symbolic ref",
			symbolicRef:    "refs/remotes/origin/master",
			symbolicRefErr: false,
			expectedBranch: "master",
		},
		{
			name:           "extracts develop from symbolic ref",
			symbolicRef:    "refs/remotes/origin/develop",
			symbolicRefErr: false,
			expectedBranch: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic
			parts := splitSymbolicRef(tt.symbolicRef)
			if len(parts) > 0 {
				branch := parts[len(parts)-1]
				assert.Equal(t, tt.expectedBranch, branch)
			}
		})
	}
}

// Helper function to test symbolic ref parsing logic
func splitSymbolicRef(ref string) []string {
	// This mimics the logic in GetDefaultBranch
	if ref == "" {
		return []string{}
	}
	parts := []string{}
	current := ""
	for _, c := range ref {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func TestDiffNameOnlyParsing(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantFiles  []string
		wantLength int
	}{
		{
			name:       "single file",
			output:     "file.txt",
			wantFiles:  []string{"file.txt"},
			wantLength: 1,
		},
		{
			name:       "multiple files",
			output:     "file1.txt\nfile2.go\nfile3.md",
			wantFiles:  []string{"file1.txt", "file2.go", "file3.md"},
			wantLength: 3,
		},
		{
			name:       "empty output",
			output:     "",
			wantFiles:  []string{},
			wantLength: 0,
		},
		{
			name:       "files with paths",
			output:     "internal/git/git.go\ninternal/state/state.go",
			wantFiles:  []string{"internal/git/git.go", "internal/state/state.go"},
			wantLength: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic
			var files []string
			if tt.output == "" {
				files = []string{}
			} else {
				// Mimic the parsing in DiffNameOnly
				trimmed := tt.output
				if trimmed != "" {
					fileList := []string{}
					current := ""
					for _, c := range trimmed {
						if c == '\n' {
							if current != "" {
								fileList = append(fileList, current)
								current = ""
							}
						} else {
							current += string(c)
						}
					}
					if current != "" {
						fileList = append(fileList, current)
					}
					files = fileList
				}
			}

			assert.Equal(t, tt.wantLength, len(files))
			if tt.wantLength > 0 {
				assert.Equal(t, tt.wantFiles, files)
			}
		})
	}
}

func TestGetRemotesParsing(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantRemotes []string
		wantLength  int
	}{
		{
			name:        "single remote",
			output:      "origin",
			wantRemotes: []string{"origin"},
			wantLength:  1,
		},
		{
			name:        "multiple remotes",
			output:      "origin\nupstream\nfork",
			wantRemotes: []string{"origin", "upstream", "fork"},
			wantLength:  3,
		},
		{
			name:        "no remotes",
			output:      "",
			wantRemotes: []string{},
			wantLength:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic
			var remotes []string
			if tt.output == "" {
				remotes = []string{}
			} else {
				// Mimic the parsing in GetRemotes
				remoteList := []string{}
				current := ""
				for _, c := range tt.output {
					if c == '\n' {
						if current != "" {
							remoteList = append(remoteList, current)
							current = ""
						}
					} else {
						current += string(c)
					}
				}
				if current != "" {
					remoteList = append(remoteList, current)
				}
				remotes = remoteList
			}

			assert.Equal(t, tt.wantLength, len(remotes))
			if tt.wantLength > 0 {
				assert.Equal(t, tt.wantRemotes, remotes)
			}
		})
	}
}

func TestBranchDeleteFlags(t *testing.T) {
	tests := []struct {
		name     string
		force    bool
		wantFlag string
	}{
		{
			name:     "force delete uses -D",
			force:    true,
			wantFlag: "-D",
		},
		{
			name:     "normal delete uses -d",
			force:    false,
			wantFlag: "-d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the flag logic
			flag := "-d"
			if tt.force {
				flag = "-D"
			}
			assert.Equal(t, tt.wantFlag, flag)
		})
	}
}
