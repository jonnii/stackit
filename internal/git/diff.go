package git

import (
	"context"
	"fmt"
	"strings"
)

// IsDiffEmpty checks if there are no differences between a branch and a base revision
func IsDiffEmpty(ctx context.Context, branchName, baseRevision string) (bool, error) {
	// Get branch revision
	branchRev, err := GetRevision(ctx, branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Check if branch revision equals base revision
	if branchRev == baseRevision {
		return true, nil
	}

	// Use git diff to check if there are any changes
	diffOutput, err := RunGitCommandWithContext(ctx, "diff", "--quiet", baseRevision, branchRev)
	if err != nil {
		// diff --quiet returns non-zero if there are differences
		return false, nil
	}

	// If diff --quiet succeeds, there are no differences
	return diffOutput == "", nil
}

// GetUnmergedFiles returns list of files with merge conflicts
func GetUnmergedFiles(ctx context.Context) ([]string, error) {
	output, err := RunGitCommandWithContext(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		// If there are no unmerged files, return empty list
		return []string{}, nil
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

// ShowDiff returns the diff between two refs with optional stat mode
func ShowDiff(ctx context.Context, left, right string, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "diff", "--no-ext-diff"}
	if stat {
		args = append(args, "--stat")
	}
	args = append(args, left, right, "--")
	return RunGitCommandWithContext(ctx, args...)
}

// ShowCommits returns commit log with optional patches/stat
// base can be empty string for trunk (will use head~), or a revision for regular branches
func ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "log"}
	if patch && stat {
		args = append(args, "--stat")
	} else if patch {
		args = append(args, "-p")
	} else {
		// Default to oneline format if no patch
		args = append(args, "--pretty=format:%h - %s")
	}

	// Always use base..head format
	// If base is empty, use head~ (parent commit) for trunk
	baseRef := base
	if base == "" {
		baseRef = head + "~"
	}
	args = append(args, fmt.Sprintf("%s..%s", baseRef, head))
	args = append(args, "--")
	return RunGitCommandWithContext(ctx, args...)
}

// GetChangedFiles returns the list of files changed between two refs
func GetChangedFiles(ctx context.Context, base, head string) ([]string, error) {
	output, err := RunGitCommandWithContext(ctx, "diff", "--name-only", base, head)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}
