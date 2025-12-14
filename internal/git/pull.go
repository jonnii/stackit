package git

import (
	"fmt"
)

// PullResult represents the result of a pull operation
type PullResult int

const (
	PullDone PullResult = iota
	PullUnneeded
	PullConflict
)

// PullBranch pulls a branch from remote
func PullBranch(remote, branchName string) (PullResult, error) {
	// Save current branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	// Switch to the branch
	if err := CheckoutBranch(branchName); err != nil {
		return PullConflict, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Fetch first
	_, _ = RunGitCommand("fetch", remote, branchName)

	// Try to merge (fast-forward only)
	_, err = RunGitCommand("merge", "--ff-only", fmt.Sprintf("%s/%s", remote, branchName))
	if err != nil {
		// Check if it's a conflict or just not fast-forwardable
		// Try to switch back to original branch
		if currentBranch != "" && currentBranch != branchName {
			_ = CheckoutBranch(currentBranch)
		}
		return PullConflict, nil
	}

	// Check if anything changed
	oldRev, _ := RunGitCommand("rev-parse", "HEAD@{1}")
	newRev, _ := RunGitCommand("rev-parse", "HEAD")
	if oldRev == newRev {
		// Switch back to original branch
		if currentBranch != "" && currentBranch != branchName {
			_ = CheckoutBranch(currentBranch)
		}
		return PullUnneeded, nil
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != branchName {
		if err := CheckoutBranch(currentBranch); err != nil {
			return PullDone, fmt.Errorf("failed to switch back to %s: %w", currentBranch, err)
		}
	}

	return PullDone, nil
}
