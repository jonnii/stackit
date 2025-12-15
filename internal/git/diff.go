package git

import (
	"fmt"
	"strings"
)

// IsDiffEmpty checks if there are no differences between a branch and a base revision
func IsDiffEmpty(branchName, baseRevision string) (bool, error) {
	// Get branch revision
	branchRev, err := GetRevision(branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Check if branch revision equals base revision
	if branchRev == baseRevision {
		return true, nil
	}

	// Use git diff to check if there are any changes
	diffOutput, err := RunGitCommand("diff", "--quiet", baseRevision, branchRev)
	if err != nil {
		// diff --quiet returns non-zero if there are differences
		return false, nil
	}

	// If diff --quiet succeeds, there are no differences
	return diffOutput == "", nil
}

// GetUnmergedFiles returns list of files with merge conflicts
func GetUnmergedFiles() ([]string, error) {
	output, err := RunGitCommand("diff", "--name-only", "--diff-filter=U")
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
func ShowDiff(left, right string, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "diff", "--no-ext-diff"}
	if stat {
		args = append(args, "--stat")
	}
	args = append(args, left, right, "--")
	return RunGitCommand(args...)
}

// ShowCommits returns commit log with optional patches/stat
func ShowCommits(base, head string, patch, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "log"}
	if patch && stat {
		args = append(args, "--stat")
	} else if patch {
		args = append(args, "-p")
	} else {
		// Default to oneline format if no patch
		args = append(args, "--pretty=format:%h - %s")
	}
	if base != "" {
		args = append(args, fmt.Sprintf("%s..%s", base, head))
	} else {
		// For trunk, show just the one commit
		args = append(args, "-1", head)
	}
	args = append(args, "--")
	return RunGitCommand(args...)
}
