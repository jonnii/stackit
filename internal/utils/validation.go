package utils

import (
	"context"
	"fmt"
	"os"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// IsInteractive checks if we're in an interactive terminal
func IsInteractive() bool {
	// Allow forcing non-interactive mode via environment variable
	if os.Getenv("STACKIT_NON_INTERACTIVE") != "" || os.Getenv("STACKIT_TEST_NO_INTERACTIVE") != "" {
		return false
	}

	// Check if stdin is a terminal
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ValidateOnBranch ensures the user is on a branch
func ValidateOnBranch(engine engine.Engine) (string, error) {
	currentBranch := engine.CurrentBranch()
	if currentBranch == nil {
		return "", fmt.Errorf("not on a branch")
	}
	return currentBranch.Name, nil
}

// CheckRebaseInProgress ensures no rebase is currently active
func CheckRebaseInProgress(ctx context.Context) error {
	if git.IsRebaseInProgress(ctx) {
		return fmt.Errorf("a rebase is already in progress. Please finish or abort it first")
	}
	return nil
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository
func HasUncommittedChanges(ctx context.Context) bool {
	// Check git status
	output, err := git.RunGitCommandWithContext(ctx, "status", "--porcelain")
	if err != nil {
		return false
	}
	return output != ""
}

// IsDemoMode returns true if STACKIT_DEMO environment variable is set
func IsDemoMode() bool {
	return os.Getenv("STACKIT_DEMO") != ""
}
