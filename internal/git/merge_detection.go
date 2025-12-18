package git

import (
	"context"
	"fmt"
)

// IsMerged checks if a branch is merged into trunk
// Uses git cherry to detect if all commits are in trunk
func IsMerged(ctx context.Context, branchName, trunkName string) (bool, error) {
	// Get merge base
	mergeBase, err := GetMergeBase(ctx, branchName, trunkName)
	if err != nil {
		return false, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get branch revision
	branchRev, err := GetRevision(ctx, branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	// If merge base equals branch revision, branch is already merged
	if mergeBase == branchRev {
		return true, nil
	}

	// Use git cherry to check if all commits are in trunk
	// git cherry <trunk> <branch> returns commits that are in branch but not in trunk
	// If empty, all commits are merged
	cherryOutput, err := RunGitCommandWithContext(ctx, "cherry", trunkName, branchName)
	if err != nil {
		// If cherry fails, fall back to simpler check
		// Check if branch tip is reachable from trunk
		_, err = RunGitCommandWithContext(ctx, "merge-base", "--is-ancestor", branchRev, trunkName)
		return err == nil, nil
	}

	// If cherry output is empty or all lines start with '-', branch is merged
	if cherryOutput == "" {
		return true, nil
	}

	// Check if all commits are marked as merged (lines starting with '-')
	lines := splitLines(cherryOutput)
	for _, line := range lines {
		if line != "" && line[0] != '-' {
			return false, nil
		}
	}

	return true, nil
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := []string{}
	current := ""
	for _, r := range s {
		if r == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
