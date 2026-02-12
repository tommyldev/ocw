package git

import (
	"fmt"
)

// Push pushes the specified branch to the remote repository
func (g *Git) Push(remote, branch string) error {
	_, err := g.run("push", remote, branch)
	if err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %w", branch, remote, err)
	}
	return nil
}

// DeleteRemoteBranch deletes a branch from the remote repository
func (g *Git) DeleteRemoteBranch(remote, branch string) error {
	// Use git push <remote> --delete <branch>
	_, err := g.run("push", remote, "--delete", branch)
	if err != nil {
		return fmt.Errorf("failed to delete remote branch %s from %s: %w", branch, remote, err)
	}
	return nil
}

// DeleteLocalBranch deletes a local branch
func (g *Git) DeleteLocalBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	_, err := g.run(args...)
	if err != nil {
		return fmt.Errorf("failed to delete local branch %s: %w", branch, err)
	}
	return nil
}
