package deps

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version (major.minor.patch).
type Version struct {
	Major int
	Minor int
	Patch int
}

// CheckResult contains the results of a dependency check.
type CheckResult struct {
	Name      string
	Installed bool
	Version   *Version
	Error     error
}

// CheckAll checks all required dependencies and returns results.
// Required: tmux, git
// Optional: gh or glab (only needed for PR/MR creation)
func CheckAll() []CheckResult {
	results := []CheckResult{
		CheckTmux(),
		CheckGit(),
	}

	// Check for gh or glab (at least one should be present for PR/MR features)
	ghResult := CheckGH()
	glabResult := CheckGLab()

	if !ghResult.Installed && !glabResult.Installed {
		results = append(results, CheckResult{
			Name:      "gh/glab",
			Installed: false,
			Error:     fmt.Errorf("neither gh nor glab found (optional - only needed for PR/MR creation)\n\nInstall one of:\n  GitHub CLI: https://cli.github.com/\n  GitLab CLI: https://gitlab.com/gitlab-org/cli"),
		})
	}

	return results
}

// CheckTmux checks if tmux is installed and meets minimum version requirements.
// Minimum version: 2.1 (for remain-on-exit support)
func CheckTmux() CheckResult {
	result := CheckResult{Name: "tmux"}

	// Check if tmux is installed
	path, err := exec.LookPath("tmux")
	if err != nil {
		result.Error = fmt.Errorf("tmux not found in PATH\n\nInstall tmux:\n  Ubuntu/Debian: sudo apt install tmux\n  macOS: brew install tmux\n  Fedora/RHEL: sudo dnf install tmux\n  Arch: sudo pacman -S tmux")
		return result
	}
	result.Installed = true

	// Get version
	cmd := exec.Command(path, "-V")
	output, err := cmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("failed to get tmux version: %w", err)
		return result
	}

	// Parse version (format: "tmux 3.2a" or "tmux 2.1")
	version, err := parseTmuxVersion(string(output))
	if err != nil {
		result.Error = fmt.Errorf("failed to parse tmux version: %w", err)
		return result
	}
	result.Version = version

	// Check minimum version (2.1)
	if version.Major < 2 || (version.Major == 2 && version.Minor < 1) {
		result.Error = fmt.Errorf("tmux version %d.%d is too old (minimum: 2.1)\n\nUpgrade tmux:\n  Ubuntu/Debian: sudo apt update && sudo apt upgrade tmux\n  macOS: brew upgrade tmux",
			version.Major, version.Minor)
	}

	return result
}

// CheckGit checks if git is installed and meets minimum version requirements.
// Minimum version: 2.15 (for worktree improvements)
func CheckGit() CheckResult {
	result := CheckResult{Name: "git"}

	// Check if git is installed
	path, err := exec.LookPath("git")
	if err != nil {
		result.Error = fmt.Errorf("git not found in PATH\n\nInstall git:\n  Ubuntu/Debian: sudo apt install git\n  macOS: brew install git\n  Fedora/RHEL: sudo dnf install git\n  Arch: sudo pacman -S git")
		return result
	}
	result.Installed = true

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("failed to get git version: %w", err)
		return result
	}

	// Parse version (format: "git version 2.34.1")
	version, err := parseGitVersion(string(output))
	if err != nil {
		result.Error = fmt.Errorf("failed to parse git version: %w", err)
		return result
	}
	result.Version = version

	// Check minimum version (2.15)
	if version.Major < 2 || (version.Major == 2 && version.Minor < 15) {
		result.Error = fmt.Errorf("git version %d.%d.%d is too old (minimum: 2.15)\n\nUpgrade git:\n  Ubuntu/Debian: sudo apt update && sudo apt upgrade git\n  macOS: brew upgrade git",
			version.Major, version.Minor, version.Patch)
	}

	return result
}

// CheckGH checks if GitHub CLI (gh) is installed.
func CheckGH() CheckResult {
	result := CheckResult{Name: "gh"}

	path, err := exec.LookPath("gh")
	if err != nil {
		return result // Not installed, but not an error (optional)
	}
	result.Installed = true

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		// Parse version (format: "gh version 2.20.2 (2022-12-15)")
		version, _ := parseGHVersion(string(output))
		result.Version = version
	}

	return result
}

// CheckGLab checks if GitLab CLI (glab) is installed.
func CheckGLab() CheckResult {
	result := CheckResult{Name: "glab"}

	path, err := exec.LookPath("glab")
	if err != nil {
		return result // Not installed, but not an error (optional)
	}
	result.Installed = true

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		// Parse version (format: "glab version 1.25.3 (2023-01-15)")
		version, _ := parseGLabVersion(string(output))
		result.Version = version
	}

	return result
}

// parseTmuxVersion parses tmux version string.
// Examples: "tmux 3.2a", "tmux 2.1", "tmux next-3.4"
func parseTmuxVersion(output string) (*Version, error) {
	// Match pattern: tmux <major>.<minor>[a-z]
	re := regexp.MustCompile(`tmux (?:next-)?(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 3 {
		return nil, fmt.Errorf("could not parse version from: %s", output)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	return &Version{
		Major: major,
		Minor: minor,
		Patch: 0,
	}, nil
}

// parseGitVersion parses git version string.
// Example: "git version 2.34.1"
func parseGitVersion(output string) (*Version, error) {
	// Match pattern: git version <major>.<minor>.<patch>
	re := regexp.MustCompile(`git version (\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 4 {
		return nil, fmt.Errorf("could not parse version from: %s", output)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// parseGHVersion parses gh version string.
// Example: "gh version 2.20.2 (2022-12-15)"
func parseGHVersion(output string) (*Version, error) {
	// Match pattern: gh version <major>.<minor>.<patch>
	re := regexp.MustCompile(`gh version (\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 4 {
		return nil, fmt.Errorf("could not parse version from: %s", output)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// parseGLabVersion parses glab version string.
// Example: "glab version 1.25.3 (2023-01-15)"
func parseGLabVersion(output string) (*Version, error) {
	// Match pattern: glab version <major>.<minor>.<patch>
	re := regexp.MustCompile(`glab version (\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 4 {
		return nil, fmt.Errorf("could not parse version from: %s", output)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// FormatResults formats dependency check results for display.
func FormatResults(results []CheckResult) string {
	var sb strings.Builder

	hasErrors := false
	for _, r := range results {
		if r.Error != nil {
			hasErrors = true
			break
		}
	}

	if !hasErrors {
		return "" // All good, no message needed
	}

	sb.WriteString("Dependency Check Failed:\n\n")

	for _, r := range results {
		if r.Error != nil {
			sb.WriteString(fmt.Sprintf("❌ %s: %v\n\n", r.Name, r.Error))
		} else if r.Installed {
			versionStr := ""
			if r.Version != nil {
				if r.Version.Patch > 0 {
					versionStr = fmt.Sprintf(" (v%d.%d.%d)", r.Version.Major, r.Version.Minor, r.Version.Patch)
				} else {
					versionStr = fmt.Sprintf(" (v%d.%d)", r.Version.Major, r.Version.Minor)
				}
			}
			sb.WriteString(fmt.Sprintf("✓ %s%s\n", r.Name, versionStr))
		}
	}

	return sb.String()
}

// HasCriticalErrors returns true if any required dependency is missing or invalid.
func HasCriticalErrors(results []CheckResult) bool {
	for _, r := range results {
		// gh/glab are optional, so skip them
		if r.Name == "gh/glab" {
			continue
		}
		if r.Error != nil {
			return true
		}
	}
	return false
}
