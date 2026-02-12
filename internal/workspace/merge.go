package workspace

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/tommyzliu/ocw/internal/state"
)

// DetectPRTool checks for gh or glab CLI tools and returns which one is available.
// It prefers gh (GitHub CLI) over glab (GitLab CLI).
// Returns: ("gh", nil), ("glab", nil), or ("", error) if neither is found.
func (m *Manager) DetectPRTool() (string, error) {
	// Check for gh first
	if _, err := exec.LookPath("gh"); err == nil {
		return "gh", nil
	}

	// Fall back to glab
	if _, err := exec.LookPath("glab"); err == nil {
		return "glab", nil
	}

	return "", fmt.Errorf("neither gh nor glab CLI tool found. Install one to create PRs:\n" +
		"  GitHub: https://cli.github.com/\n" +
		"  GitLab: https://gitlab.com/gitlab-org/cli")
}

// PushBranch pushes the branch for the specified instance to the remote repository.
// It uses "origin" as the default remote.
func (m *Manager) PushBranch(instanceID string) error {
	// Load state to get instance details
	stateData, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find instance
	var instance *state.Instance
	for i := range stateData.Instances {
		if stateData.Instances[i].ID == instanceID {
			instance = &stateData.Instances[i]
			break
		}
	}

	if instance == nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	// Push the branch
	if err := m.git.Push("origin", instance.Branch); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", instance.Branch, err)
	}

	return nil
}

// CreatePR creates a pull request for the specified instance using gh or glab CLI.
// It detects which tool is available and uses the appropriate command.
// Returns the PR URL on success.
func (m *Manager) CreatePR(instanceID, title, body string) (string, error) {
	// Load state to get instance details
	stateData, err := m.store.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	// Find instance
	var instance *state.Instance
	for i := range stateData.Instances {
		if stateData.Instances[i].ID == instanceID {
			instance = &stateData.Instances[i]
			break
		}
	}

	if instance == nil {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}

	// Detect which PR tool is available
	tool, err := m.DetectPRTool()
	if err != nil {
		return "", err
	}

	var prURL string
	switch tool {
	case "gh":
		prURL, err = m.createGitHubPR(instance.Branch, instance.BaseBranch, title, body)
	case "glab":
		prURL, err = m.createGitLabMR(instance.Branch, instance.BaseBranch, title, body)
	default:
		return "", fmt.Errorf("unsupported PR tool: %s", tool)
	}

	if err != nil {
		return "", err
	}

	// Update instance with PR URL and status
	if err := m.store.UpdateInstance(instanceID, func(inst *state.Instance) {
		inst.PRUrl = prURL
		inst.Status = "merged"
	}); err != nil {
		return prURL, fmt.Errorf("PR created at %s but failed to update state: %w", prURL, err)
	}

	return prURL, nil
}

// createGitHubPR creates a GitHub pull request using the gh CLI.
func (m *Manager) createGitHubPR(branch, base, title, body string) (string, error) {
	args := []string{
		"pr", "create",
		"--title", title,
		"--base", base,
	}

	if body != "" {
		args = append(args, "--body", body)
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = m.repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create failed: %w\nOutput: %s", err, string(output))
	}

	// The gh CLI returns the PR URL as the last line of output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output from gh pr create")
	}

	prURL := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(prURL, "http") {
		// If the last line isn't a URL, try to find one in the output
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], "http") {
				prURL = strings.TrimSpace(lines[i])
				break
			}
		}
	}

	return prURL, nil
}

// createGitLabMR creates a GitLab merge request using the glab CLI.
func (m *Manager) createGitLabMR(branch, base, title, body string) (string, error) {
	args := []string{
		"mr", "create",
		"--title", title,
		"--target-branch", base,
	}

	if body != "" {
		args = append(args, "--description", body)
	}

	cmd := exec.Command("glab", args...)
	cmd.Dir = m.repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("glab mr create failed: %w\nOutput: %s", err, string(output))
	}

	// The glab CLI returns the MR URL in the output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output from glab mr create")
	}

	// Find the URL in the output
	mrURL := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "http") {
			mrURL = strings.TrimSpace(lines[i])
			break
		}
	}

	if mrURL == "" {
		return "", fmt.Errorf("could not find MR URL in glab output: %s", string(output))
	}

	return mrURL, nil
}
