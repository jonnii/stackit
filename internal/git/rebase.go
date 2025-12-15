package git

import (
	"fmt"
	"os"
	"os/exec"
)

// RebaseResult represents the result of a rebase operation
type RebaseResult int

const (
	RebaseDone RebaseResult = iota
	RebaseConflict
)

// Rebase rebases a branch onto another branch
// onto is the branch name to rebase onto (parent branch)
// from is the base revision (old parent branch revision)
func Rebase(branchName, onto, from string) (RebaseResult, error) {
	// Save current branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	// Checkout the branch to rebase
	if err := CheckoutBranch(branchName); err != nil {
		return RebaseConflict, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Perform rebase using git rebase --onto
	// git rebase --onto <onto> <from>
	// This rebases commits from <from>..HEAD onto <onto>
	_, err = RunGitCommand("rebase", "--onto", onto, from)
	if err != nil {
		// Check if rebase is in progress (conflict)
		if IsRebaseInProgress() {
			// Rebase is in progress, switch back
			if currentBranch != "" && currentBranch != branchName {
				_ = CheckoutBranch(currentBranch)
			}
			return RebaseConflict, nil
		}
		// Switch back to original branch
		if currentBranch != "" && currentBranch != branchName {
			_ = CheckoutBranch(currentBranch)
		}
		return RebaseConflict, nil
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != branchName {
		if err := CheckoutBranch(currentBranch); err != nil {
			return RebaseDone, fmt.Errorf("failed to switch back to %s: %w", currentBranch, err)
		}
	}

	return RebaseDone, nil
}

// IsRebaseInProgress checks if a rebase is currently in progress
func IsRebaseInProgress() bool {
	_, err := RunGitCommand("rev-parse", "--git-path", "REBASE_HEAD")
	return err == nil
}

// RebaseContinue continues an in-progress rebase
func RebaseContinue() (RebaseResult, error) {
	// Set GIT_EDITOR to 'true' to skip editor (like Charcoal does)
	env := os.Environ()
	env = append(env, "GIT_EDITOR=true")

	cmd := exec.Command("git", "rebase", "--continue")
	cmd.Env = env
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		// Check if rebase is still in progress (another conflict)
		if IsRebaseInProgress() {
			return RebaseConflict, nil
		}
		return RebaseConflict, fmt.Errorf("rebase continue failed: %w", err)
	}

	return RebaseDone, nil
}

// GetRebaseHead returns the commit being rebased (REBASE_HEAD)
func GetRebaseHead() (string, error) {
	return RunGitCommand("rev-parse", "REBASE_HEAD")
}
