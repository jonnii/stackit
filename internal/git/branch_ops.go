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

// DeleteBranch deletes a branch
func DeleteBranch(ctx context.Context, branchName string) error {
	_, err := RunGitCommandWithContext(ctx, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}
	return nil
}
