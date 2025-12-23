package git

import (
	"context"
	"fmt"
	"strings"
)

// Branch represents a branch for git operations. This interface allows
// the git package to work with branches without importing the engine package,
// avoiding circular dependencies.
type Branch interface {
	GetName() string
}

// branchName is a simple implementation of Branch for use within the git package
type branchName string

func (b branchName) GetName() string {
	return string(b)
}

// NewBranch creates a Branch from a string name. This is useful for converting
// string branch names to Branch objects within the git package.
func NewBranch(name string) Branch {
	return branchName(name)
}

// CreateAndCheckoutBranch creates and checks out a new branch
func CreateAndCheckoutBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
	_, err := RunGitCommandWithContext(ctx, "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
	}
	return nil
}

// CheckoutBranch checks out an existing branch
func CheckoutBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
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
func DeleteBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
	_, err := RunGitCommandWithContext(ctx, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
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
