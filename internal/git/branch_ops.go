package git

import (
	"context"
	"fmt"
	"strings"
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
		// If it's already used by another worktree, try checking out detached
		if strings.Contains(err.Error(), "already used by worktree") {
			return CheckoutDetached(ctx, branchName)
		}
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
