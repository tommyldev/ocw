package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DiffStat represents statistics from a git diff
type DiffStat struct {
	FilesChanged int
	Insertions   int
	Deletions    int
	Summary      string
}

// DiffFile represents a single file in a diff with its change status
type DiffFile struct {
	Status string // M (Modified), A (Added), D (Deleted), R (Renamed), etc.
	Path   string
}

// DiffStat returns diff statistics between the working tree and the specified base
func (g *Git) DiffStat(base string) (DiffStat, error) {
	output, err := g.run("diff", "--stat", base)
	if err != nil {
		return DiffStat{}, fmt.Errorf("failed to get diff stat: %w", err)
	}

	return parseDiffStat(output), nil
}

// DiffStatBranch returns diff statistics between two branches
func (g *Git) DiffStatBranch(branch, base string) (DiffStat, error) {
	output, err := g.run("diff", "--stat", fmt.Sprintf("%s..%s", base, branch))
	if err != nil {
		return DiffStat{}, fmt.Errorf("failed to get diff stat between branches: %w", err)
	}

	return parseDiffStat(output), nil
}

// parseDiffStat parses the output from git diff --stat
// Example output:
//
//	file1.go | 10 ++++++++++
//	file2.go | 5 -----
//	2 files changed, 10 insertions(+), 5 deletions(-)
func parseDiffStat(output string) DiffStat {
	stat := DiffStat{}

	if output == "" {
		return stat
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return stat
	}

	// The summary line is typically the last non-empty line
	summaryLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			summaryLine = strings.TrimSpace(lines[i])
			break
		}
	}

	stat.Summary = summaryLine

	// Parse summary line: "3 files changed, 50 insertions(+), 10 deletions(-)"
	// Regex patterns for different parts
	filesChangedRe := regexp.MustCompile(`(\d+) files? changed`)
	insertionsRe := regexp.MustCompile(`(\d+) insertions?\(\+\)`)
	deletionsRe := regexp.MustCompile(`(\d+) deletions?\(-\)`)

	if matches := filesChangedRe.FindStringSubmatch(summaryLine); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			stat.FilesChanged = val
		}
	}

	if matches := insertionsRe.FindStringSubmatch(summaryLine); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			stat.Insertions = val
		}
	}

	if matches := deletionsRe.FindStringSubmatch(summaryLine); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			stat.Deletions = val
		}
	}

	return stat
}

// DiffFiles returns a list of files changed between the working tree and the specified base
func (g *Git) DiffFiles(base string) ([]DiffFile, error) {
	output, err := g.run("diff", "--name-status", base)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	return parseDiffFiles(output), nil
}

// parseDiffFiles parses the output from git diff --name-status
// Example output:
// M	file1.go
// A	file2.go
// D	file3.go
func parseDiffFiles(output string) []DiffFile {
	var files []DiffFile

	if output == "" {
		return files
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by tab or whitespace
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			files = append(files, DiffFile{
				Status: parts[0],
				Path:   strings.Join(parts[1:], " "), // Join in case path has spaces
			})
		}
	}

	return files
}
