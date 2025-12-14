package git

import (
	"fmt"
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
