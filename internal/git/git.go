package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Git represents a Git repository and provides methods to interact with it via CLI
type Git struct {
	repoPath string
}

// NewGit creates a new Git instance for the specified repository path
func NewGit(repoPath string) *Git {
	return &Git{repoPath: repoPath}
}

// run executes a git command with the given arguments in the repository
func (g *Git) run(args ...string) (string, error) {
	cmdArgs := append([]string{"-C", g.repoPath}, args...)
	cmd := exec.Command("git", cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepo checks if the specified path is a git repository
func (g *Git) IsGitRepo() bool {
	_, err := g.run("rev-parse", "--git-dir")
	return err == nil
}

// GetDefaultBranch detects the default branch (main or master)
func (g *Git) GetDefaultBranch() (string, error) {
	// Try to get the default branch from remote HEAD
	output, err := g.run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		// Output format: refs/remotes/origin/main
		parts := strings.Split(output, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if main exists, otherwise master
	if g.BranchExists("main") {
		return "main", nil
	}

	if g.BranchExists("master") {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

// GetCurrentBranch returns the current branch name
func (g *Git) GetCurrentBranch() (string, error) {
	output, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return output, nil
}

// GetHeadSHA returns the SHA of the current HEAD
func (g *Git) GetHeadSHA() (string, error) {
	output, err := g.run("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD SHA: %w", err)
	}
	return output, nil
}

// BranchExists checks if a branch exists locally
func (g *Git) BranchExists(branch string) bool {
	_, err := g.run("show-ref", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	return err == nil
}

// GetRemotes returns a list of configured remotes
func (g *Git) GetRemotes() ([]string, error) {
	output, err := g.run("remote")
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	remotes := strings.Split(output, "\n")
	return remotes, nil
}
