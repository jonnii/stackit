package git

import (
	"fmt"
)

// CreateAndCheckoutBranch creates and checks out a new branch
func CreateAndCheckoutBranch(branchName string) error {
	_, err := RunGitCommand("checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
	}
	return nil
}

// CheckoutBranch checks out an existing branch
func CheckoutBranch(branchName string) error {
	_, err := RunGitCommand("checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(branchName string) error {
	_, err := RunGitCommand("branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}
	return nil
}
