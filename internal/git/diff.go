package git

import (
	"context"
	"fmt"
	"strings"
)

// IsDiffEmpty checks if there are no differences between a branch and a base revision
func IsDiffEmpty(ctx context.Context, branchName, baseRevision string) (bool, error) {
	branchRev, err := GetRevision(branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	if branchRev == baseRevision {
		return true, nil
	}

	_, err = RunGitCommandWithContext(ctx, "diff", "--quiet", baseRevision, branchRev)
	return err == nil, nil
}

// GetUnmergedFiles returns list of files with merge conflicts
func GetUnmergedFiles(ctx context.Context) ([]string, error) {
	output, err := RunGitCommandWithContext(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return []string{}, nil //nolint:nilerr
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
	switch {
	case patch && stat:
		args = append(args, "--stat")
	case patch:
		args = append(args, "-p")
	default:
		args = append(args, "--pretty=format:%h - %s")
	}

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
