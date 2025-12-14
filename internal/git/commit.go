package git

import (
	"fmt"
	"os"
	"os/exec"
)

// Commit creates a commit with the given message
// If verbose > 0, shows unified diff in commit message template
func Commit(message string, verbose int) error {
	// If verbose > 0, we need to use git commit with -v flag
	// For verbose > 1, we'd need -vv but git commit only supports single -v
	args := []string{"commit"}
	
	if verbose > 0 {
		args = append(args, "-v")
	}
	
	if message != "" {
		args = append(args, "-m", message)
	}
	// If no message provided, git will open editor (interactive)
	
	// Use exec.Command directly to allow for interactive commit if needed
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// GetStagedDiff returns the unified diff of staged changes
func GetStagedDiff() (string, error) {
	output, err := RunGitCommand("diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return output, nil
}

// GetUnstagedDiff returns the unified diff of unstaged changes
func GetUnstagedDiff() (string, error) {
	output, err := RunGitCommand("diff")
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged diff: %w", err)
	}
	return output, nil
}
