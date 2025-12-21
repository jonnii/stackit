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
	// Save current branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	// Switch to the branch
	if err := CheckoutBranch(ctx, branchName); err != nil {
		return PullConflict, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Fetch first
	_, _ = RunGitCommandWithContext(ctx, "fetch", remote, branchName)

	// Try to merge (fast-forward only)
	_, err = RunGitCommandWithContext(ctx, "merge", "--ff-only", fmt.Sprintf("%s/%s", remote, branchName))
	if err != nil {
		// Check if it's a conflict or just not fast-forwardable
		// Try to switch back to original branch
		if currentBranch != "" && currentBranch != branchName {
			_ = CheckoutBranch(ctx, currentBranch)
		}
		return PullConflict, nil
	}

	// Check if anything changed
	oldRev, _ := RunGitCommandWithContext(ctx, "rev-parse", "HEAD@{1}")
	newRev, _ := RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	if oldRev == newRev {
		// Switch back to original branch
		if currentBranch != "" && currentBranch != branchName {
			_ = CheckoutBranch(ctx, currentBranch)
		}
		return PullUnneeded, nil
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != branchName {
		if err := CheckoutBranch(ctx, currentBranch); err != nil {
			return PullDone, fmt.Errorf("failed to switch back to %s: %w", currentBranch, err)
		}
	}

	return PullDone, nil
}
