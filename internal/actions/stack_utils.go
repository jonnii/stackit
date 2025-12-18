package actions

import (
	"stackit.dev/stackit/internal/engine"
)

// SortBranchesTopologically sorts branches so parents come before children.
// This ensures correct restack order (bottom of stack first).
func SortBranchesTopologically(branches []string, eng engine.BranchReader) []string {
	if len(branches) == 0 {
		return branches
	}

	// Build a set for quick lookup
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	// Calculate depth for each branch (distance from trunk)
	depths := make(map[string]int)
	var getDepth func(branch string) int
	getDepth = func(branch string) int {
		if depth, ok := depths[branch]; ok {
			return depth
		}
		if eng.IsTrunk(branch) {
			depths[branch] = 0
			return 0
		}
		parent := eng.GetParent(branch)
		if parent == "" || eng.IsTrunk(parent) {
			depths[branch] = 1
			return 1
		}
		depths[branch] = getDepth(parent) + 1
		return depths[branch]
	}

	// Calculate depth for all branches
	for _, branch := range branches {
		getDepth(branch)
	}

	// Sort by depth (parents first, then children)
	result := make([]string, len(branches))
	copy(result, branches)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if depths[result[i]] > depths[result[j]] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetUpstack returns all branches upstack from the given branch (descendants)
func GetUpstack(eng engine.BranchReader, branchName string) []string {
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	return eng.GetRelativeStack(branchName, scope)
}

// GetDownstack returns all branches downstack from the given branch (ancestors)
func GetDownstack(eng engine.BranchReader, branchName string) []string {
	scope := engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	return eng.GetRelativeStack(branchName, scope)
}

// GetFullStack returns the entire stack containing the given branch
func GetFullStack(eng engine.BranchReader, branchName string) []string {
	scope := engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: true,
	}
	return eng.GetRelativeStack(branchName, scope)
}
