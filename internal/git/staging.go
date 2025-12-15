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
	// Note: This requires interactive input, so we'll use exec.Command directly
	// For now, we'll use git add -p which requires user interaction
	// This will be handled at the CLI level with proper stdin/stdout
	_, err := RunGitCommand("add", "-p")
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

// HasUnstagedChanges checks if there are unstaged changes
func HasUnstagedChanges() (bool, error) {
	output, err := RunGitCommand("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	
	// Parse porcelain output - if any line doesn't start with space-space, there are unstaged changes
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Porcelain format: XY filename
		// If X is not space, there are staged changes
		// If Y is not space, there are unstaged changes
		if len(line) >= 2 && line[1] != ' ' {
			return true, nil
		}
	}
	return false, nil
}

// HasUntrackedFiles checks if there are untracked files
func HasUntrackedFiles() (bool, error) {
	output, err := RunGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return false, fmt.Errorf("failed to check untracked files: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

