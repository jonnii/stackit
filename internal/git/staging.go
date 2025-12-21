package git

import (
	"context"
	"fmt"
	"strings"
)

// StageAll stages all changes including untracked files
func StageAll(ctx context.Context) error {
	_, err := RunGitCommandWithContext(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

// AddAll is an alias for StageAll (for compatibility with continue command)
func AddAll(ctx context.Context) error {
	return StageAll(ctx)
}

// StageTracked stages updates to tracked files only
func StageTracked(ctx context.Context) error {
	_, err := RunGitCommandWithContext(ctx, "add", "-u")
	if err != nil {
		return fmt.Errorf("failed to stage tracked changes: %w", err)
	}
	return nil
}

// StagePatch performs interactive patch staging
func StagePatch(_ context.Context) error {
	// Use interactive mode so stdin/stdout/stderr are connected to the terminal
	// Note: RunGitCommandInteractive doesn't take context yet, but it's okay for now
	err := RunGitCommandInteractive("add", "-p")
	if err != nil {
		return fmt.Errorf("failed to stage patch: %w", err)
	}
	return nil
}

// HasStagedChanges checks if there are staged changes
func HasStagedChanges(ctx context.Context) (bool, error) {
	output, err := RunGitCommandWithContext(ctx, "diff", "--cached", "--shortstat")
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// HasUnstagedChanges checks if there are unstaged changes to tracked files
func HasUnstagedChanges(ctx context.Context) (bool, error) {
	// Use git diff to check for unstaged changes to tracked files
	// This is more reliable than parsing porcelain output which gets trimmed
	output, err := RunGitCommandWithContext(ctx, "diff", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// HasUntrackedFiles checks if there are untracked files
func HasUntrackedFiles(ctx context.Context) (bool, error) {
	output, err := RunGitCommandWithContext(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return false, fmt.Errorf("failed to check untracked files: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}
