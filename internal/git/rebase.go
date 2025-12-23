package git

import (
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5/plumbing"
)

// RebaseResult represents the result of a rebase operation
type RebaseResult int

const (
	// RebaseDone indicates the rebase was successful
	RebaseDone RebaseResult = iota
	// RebaseConflict indicates a conflict occurred during rebase
	RebaseConflict
)

// Rebase rebases a branch onto another branch
// onto is the branch name to rebase onto (parent branch)
// from is the base revision (old parent branch revision)
func Rebase(ctx context.Context, branchName, onto, from string) (RebaseResult, error) {
	// Save current branch/detached HEAD
	currentBranch, err := GetCurrentBranch()
	var currentRev string
	if err != nil {
		currentBranch = ""
		currentRev, _ = RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	}

	// Get the SHA of the branch we want to rebase
	branchRev, err := RunGitCommandWithContext(ctx, "rev-parse", branchName)
	if err != nil {
		return RebaseConflict, fmt.Errorf("failed to get revision for %s: %w", branchName, err)
	}

	// Perform rebase using detached HEAD to avoid "already used by worktree" errors
	// git rebase --onto <onto> <from> <branchRev>
	// This will result in a detached HEAD at the new rebased commit
	_, err = RunGitCommandWithContext(ctx, "rebase", "--onto", onto, from, branchRev)
	if err != nil {
		// Check if rebase is in progress (conflict)
		if IsRebaseInProgress(ctx) {
			return RebaseConflict, nil
		}
		// Try to abort rebase if it failed for other reasons
		_, _ = RunGitCommandWithContext(ctx, "rebase", "--abort")

		// Restore original state
		if currentBranch != "" {
			_ = CheckoutBranch(ctx, NewBranch(currentBranch))
		} else if currentRev != "" {
			_ = CheckoutDetached(ctx, currentRev)
		}
		return RebaseConflict, nil
	}

	// Get the new rebased SHA
	newRev, err := RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	if err != nil {
		return RebaseConflict, fmt.Errorf("failed to get new revision after rebase: %w", err)
	}

	// Update the branch reference to the new rebased commit
	_, err = RunGitCommandWithContext(ctx, "update-ref", "refs/heads/"+branchName, newRev)
	if err != nil {
		return RebaseConflict, fmt.Errorf("failed to update branch reference %s: %w", branchName, err)
	}

	// Restore original state
	if currentBranch != "" {
		if err := CheckoutBranch(ctx, NewBranch(currentBranch)); err != nil {
			// If original branch is now used elsewhere (unlikely but possible), checkout detached
			_ = CheckoutDetached(ctx, currentBranch)
		}
	} else if currentRev != "" {
		_ = CheckoutDetached(ctx, currentRev)
	}

	return RebaseDone, nil
}

// IsRebaseInProgress checks if a rebase is currently in progress
func IsRebaseInProgress(ctx context.Context) bool {
	// Check for .git/rebase-merge or .git/rebase-apply directories
	// This is more reliable than checking REBASE_HEAD which can persist after rebase
	output, err := RunGitCommandWithContext(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return false
	}

	gitDir := output
	// Check for interactive rebase
	if _, err := os.Stat(gitDir + "/rebase-merge"); err == nil {
		return true
	}
	// Check for non-interactive rebase
	if _, err := os.Stat(gitDir + "/rebase-apply"); err == nil {
		return true
	}
	return false
}

// RebaseContinue continues an in-progress rebase
func RebaseContinue(ctx context.Context) (RebaseResult, error) {
	// Use RunGitCommandWithContext, but we need to set GIT_EDITOR=true
	// Since RunGitCommandWithContext doesn't support custom env yet, we'll use a hack or update the runner.
	// Actually, let's just use exec.Command for now if we need custom env, but the goal is to use the runner.
	// I'll stick to a simpler version for now.

	_, err := RunGitCommandWithContext(ctx, "-c", "core.editor=true", "rebase", "--continue")
	if err != nil {
		// Check if rebase is still in progress (another conflict)
		if IsRebaseInProgress(ctx) {
			return RebaseConflict, nil
		}
		return RebaseConflict, fmt.Errorf("rebase continue failed: %w", err)
	}

	return RebaseDone, nil
}

// RebaseAbort aborts an in-progress rebase
func RebaseAbort(ctx context.Context) error {
	_, err := RunGitCommandWithContext(ctx, "rebase", "--abort")
	if err != nil {
		return fmt.Errorf("rebase abort failed: %w", err)
	}
	return nil
}

// GetRebaseHead returns the commit being rebased (REBASE_HEAD)
func GetRebaseHead() (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	// Try the standard rebase head refs
	refs := []plumbing.ReferenceName{
		"refs/rebase-merge/head",
		"refs/rebase-apply/head",
		"REBASE_HEAD",
	}

	for _, refName := range refs {
		ref, err := repo.Reference(refName, true)
		if err == nil {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("rebase head not found")
}
