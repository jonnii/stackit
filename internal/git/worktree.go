package git

import (
	"context"
	"fmt"
)

// AddWorktree adds a new worktree at the specified path
// branch is the branch to checkout in the worktree
// if detach is true, the worktree will be in detached HEAD state
func AddWorktree(ctx context.Context, path string, branch string, detach bool) error {
	args := []string{"worktree", "add"}
	if detach {
		args = append(args, "--detach")
	}
	args = append(args, path)
	if branch != "" {
		args = append(args, branch)
	}

	_, err := RunGitCommandWithContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to add worktree at %s: %w", path, err)
	}
	return nil
}

// RemoveWorktree removes the worktree at the specified path
func RemoveWorktree(ctx context.Context, path string) error {
	_, err := RunGitCommandWithContext(ctx, "worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %w", path, err)
	}
	return nil
}

// ListWorktrees returns a list of worktrees
func ListWorktrees(ctx context.Context) ([]string, error) {
	lines, err := RunGitCommandLinesWithContext(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []string
	for _, line := range lines {
		if len(line) > 9 && line[:9] == "worktree " {
			worktrees = append(worktrees, line[9:])
		}
	}
	return worktrees, nil
}
