package git

import (
	"fmt"
	"strings"
)

// StageAll stages all changes including untracked files
func StageAll() error {
	_, err := RunGitCommand("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

// AddAll is an alias for StageAll (for compatibility with continue command)
func AddAll() error {
	return StageAll()
}

// StageTracked stages updates to tracked files only
func StageTracked() error {
	_, err := RunGitCommand("add", "-u")
	if err != nil {
		return fmt.Errorf("failed to stage tracked changes: %w", err)
	}
	return nil
}

// StagePatch performs interactive patch staging
func StagePatch() error {
	// Use interactive mode so stdin/stdout/stderr are connected to the terminal
	err := RunGitCommandInteractive("add", "-p")
	if err != nil {
		return fmt.Errorf("failed to stage patch: %w", err)
	}
	return nil
}

// HasStagedChanges checks if there are staged changes
func HasStagedChanges() (bool, error) {
	output, err := RunGitCommand("diff", "--cached", "--shortstat")
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// HasUnstagedChanges checks if there are unstaged changes to tracked files
func HasUnstagedChanges() (bool, error) {
	// Use git diff to check for unstaged changes to tracked files
	// This is more reliable than parsing porcelain output which gets trimmed
	output, err := RunGitCommand("diff", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// HasUntrackedFiles checks if there are untracked files
func HasUntrackedFiles() (bool, error) {
	output, err := RunGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return false, fmt.Errorf("failed to check untracked files: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}
