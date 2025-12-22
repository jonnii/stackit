package git

import (
	"context"
	"fmt"
)

// PullResult represents the result of a pull operation
type PullResult int

const (
	// PullDone indicates the pull was successful
	PullDone PullResult = iota
	// PullUnneeded indicates no pull was needed
	PullUnneeded
	// PullConflict indicates a conflict occurred during pull
	PullConflict
)

// PullBranch pulls a branch from remote
func PullBranch(ctx context.Context, remote, branchName string) (PullResult, error) {
	// Save current branch/detached HEAD
	currentBranch, err := GetCurrentBranch()
	var currentRev string
	if err != nil {
		currentBranch = ""
		currentRev, _ = RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	}

	// Get the SHA of the local branch
	oldRev, err := RunGitCommandWithContext(ctx, "rev-parse", branchName)
	if err != nil {
		return PullConflict, fmt.Errorf("failed to get local revision for %s: %w", branchName, err)
	}

	// Fetch first
	_, _ = RunGitCommandWithContext(ctx, "fetch", remote, branchName)

	// Get the SHA of the remote branch
	remoteRev, err := RunGitCommandWithContext(ctx, "rev-parse", fmt.Sprintf("%s/%s", remote, branchName))
	if err != nil {
		// If we can't get remote rev, we can't pull, but it might just be because there's no remote
		return PullUnneeded, nil //nolint:nilerr
	}

	if oldRev == remoteRev {
		return PullUnneeded, nil
	}

	// Check if it's a fast-forward
	isAncestor, err := IsAncestor(oldRev, remoteRev)
	if err != nil || !isAncestor {
		return PullConflict, nil //nolint:nilerr
	}

	// Update the local branch reference to the remote commit (fast-forward)
	_, err = RunGitCommandWithContext(ctx, "update-ref", "refs/heads/"+branchName, remoteRev)
	if err != nil {
		return PullConflict, fmt.Errorf("failed to update local branch %s to %s: %w", branchName, remoteRev, err)
	}

	// If we are currently ON this branch in this worktree, we need to update HEAD
	if currentBranch == branchName {
		_ = CheckoutBranch(ctx, branchName)
	} else if currentRev != "" {
		_ = CheckoutDetached(ctx, currentRev)
	}

	return PullDone, nil
}
