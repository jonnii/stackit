package git

import (
	"context"
	"fmt"
)

// CreateAndCheckoutBranch creates and checks out a new branch
func CreateAndCheckoutBranch(ctx context.Context, branchName string) error {
	_, err := RunGitCommandWithContext(ctx, "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
	}
	return nil
}

// CheckoutBranch checks out an existing branch
func CheckoutBranch(ctx context.Context, branchName string) error {
	_, err := RunGitCommandWithContext(ctx, "checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

// CheckoutDetached checks out a revision in detached HEAD state
func CheckoutDetached(ctx context.Context, rev string) error {
	_, err := RunGitCommandWithContext(ctx, "checkout", "--detach", rev)
	if err != nil {
		return fmt.Errorf("failed to checkout %s in detached state: %w", rev, err)
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(ctx context.Context, branchName string) error {
	_, err := RunGitCommandWithContext(ctx, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}
	return nil
}

// RenameBranch renames a branch
func RenameBranch(ctx context.Context, oldName, newName string) error {
	_, err := RunGitCommandWithContext(ctx, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename branch %s to %s: %w", oldName, newName, err)
	}
	return nil
}

// UpdateBranchRef updates a branch reference to point to a new commit
func UpdateBranchRef(branchName, commitSHA string) error {
	_, err := RunGitCommandWithContext(context.Background(), "update-ref", "refs/heads/"+branchName, commitSHA)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}
	return nil
}

// MergeAbort aborts an in-progress merge
func MergeAbort(ctx context.Context) error {
	_, err := RunGitCommandWithContext(ctx, "merge", "--abort")
	if err != nil {
		return fmt.Errorf("merge abort failed: %w", err)
	}
	return nil
}

// StashPush pushes current changes to the stash
func StashPush(ctx context.Context, message string) (string, error) {
	args := []string{"stash", "push", "-u"}
	if message != "" {
		args = append(args, "-m", message)
	}
	output, err := RunGitCommandWithContext(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("stash push failed: %w", err)
	}
	return output, nil
}

// StashPop pops the most recent stash
func StashPop(ctx context.Context) error {
	_, err := RunGitCommandWithContext(ctx, "stash", "pop")
	if err != nil {
		return fmt.Errorf("stash pop failed: %w", err)
	}
	return nil
}
