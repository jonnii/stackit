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
