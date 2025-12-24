package doctor

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// checkStackState performs stack state and metadata integrity checks
func checkStackState(eng engine.Engine, splog *tui.Splog, warnings []string, errors []string, fix bool) ([]string, []string) {
	// Get all branches
	allBranches, err := git.GetAllBranchNames()
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get branch names: %v", err))
		splog.Error("  failed to get branch names: %v", err)
		return warnings, errors
	}

	// Get all metadata refs
	metadataRefs, err := git.GetMetadataRefList()
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get metadata refs: %v", err))
		splog.Error("  failed to get metadata refs: %v", err)
		return warnings, errors
	}

	// Check for orphaned metadata (metadata for branches that don't exist)
	branchSet := make(map[string]bool)
	for _, branch := range allBranches {
		branchSet[branch] = true
	}

	orphanedCount := 0
	prunedCount := 0
	for branchName := range metadataRefs {
		if !branchSet[branchName] {
			orphanedCount++
			if fix {
				if err := git.DeleteMetadataRef(branchName); err != nil {
					splog.Error("  Failed to prune orphaned metadata for %s: %v", branchName, err)
					warnings = append(warnings, fmt.Sprintf("orphaned metadata found for deleted branch '%s' (fix failed)", branchName))
				} else {
					splog.Info("  ✅ Pruned orphaned metadata for deleted branch %s", tui.ColorBranchName(branchName, false))
					prunedCount++
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("orphaned metadata found for deleted branch '%s'", branchName))
			}
		}
	}

	if orphanedCount > 0 {
		if fix {
			if prunedCount == orphanedCount {
				splog.Info("  ✅ All %d orphaned metadata ref(s) pruned", prunedCount)
			} else {
				splog.Warn("  Found %d orphaned metadata ref(s), pruned %d", orphanedCount, prunedCount)
			}
		} else {
			splog.Warn("  Found %d orphaned metadata ref(s) (run 'stackit doctor --fix' to prune)", orphanedCount)
		}
	} else {
		splog.Info("  ✅ No orphaned metadata found")
	}

	// Check for corrupted metadata
	corruptedCount := 0
	for branchName := range metadataRefs {
		meta, err := git.ReadMetadataRef(branchName)
		if err != nil {
			corruptedCount++
			errors = append(errors, fmt.Sprintf("corrupted metadata for branch '%s': %v", branchName, err))
		} else if meta != nil {
			// Validate that if parent is set, it's not empty
			if meta.ParentBranchName != nil && *meta.ParentBranchName == "" {
				corruptedCount++
				errors = append(errors, fmt.Sprintf("invalid metadata for branch '%s': parent branch name is empty", branchName))
			}
		}
	}

	if corruptedCount > 0 {
		splog.Error("  Found %d corrupted metadata ref(s)", corruptedCount)
	} else {
		splog.Info("  ✅ Metadata integrity check passed")
	}

	// Check for cycles in the stack graph
	cycles := detectCycles(eng)
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			errors = append(errors, fmt.Sprintf("cycle detected in stack graph: %s", strings.Join(cycle, " -> ")))
		}
		splog.Error("  Found %d cycle(s) in stack graph", len(cycles))
	} else {
		splog.Info("  ✅ No cycles detected in stack graph")
	}

	// Check for missing parent branches
	missingParents := checkMissingParents(eng, allBranches)
	if len(missingParents) > 0 {
		for _, branch := range missingParents {
			branchObj := eng.GetBranch(branch)
			parent := eng.GetParent(branchObj)
			parentName := "unknown"
			if parent != nil {
				parentName = parent.Name
			}
			warnings = append(warnings, fmt.Sprintf("branch '%s' has parent '%s' that does not exist", branch, parentName))
		}
		splog.Warn("  Found %d branch(es) with missing parents", len(missingParents))
	} else {
		splog.Info("  ✅ All parent branches exist")
	}

	return warnings, errors
}

// detectCycles detects cycles in the branch parent graph using DFS
func detectCycles(eng engine.Engine) [][]string {
	var cycles [][]string
	allBranches := eng.AllBranches()
	branchNames := make([]string, len(allBranches))
	for i, b := range allBranches {
		branchNames[i] = b.Name
	}
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parentMap := make(map[string]string)
	trunk := eng.Trunk()
	trunkName := trunk.Name

	// Build parent map
	for _, branch := range allBranches {
		branchName := branch.Name
		parent := eng.GetParent(branch)
		if parent != nil && parent.Name != trunkName {
			parentMap[branchName] = parent.Name
		}
	}

	var dfs func(string, []string)
	dfs = func(branch string, path []string) {
		if recStack[branch] {
			// Found a cycle - find where the cycle starts in the path
			cycleStart := -1
			for i, b := range path {
				if b == branch {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				// Extract the cycle: from first occurrence to current
				cycle := make([]string, len(path)-cycleStart+1)
				copy(cycle, path[cycleStart:])
				cycle[len(cycle)-1] = branch
				cycles = append(cycles, cycle)
			}
			return
		}

		if visited[branch] {
			// Already fully explored this branch
			return
		}

		visited[branch] = true
		recStack[branch] = true

		// Follow parent if it exists
		if parent, hasParent := parentMap[branch]; hasParent {
			dfs(parent, append(path, branch))
		}

		recStack[branch] = false
	}

	// Run DFS on all branches
	for _, branchName := range branchNames {
		if branchName != trunkName && !visited[branchName] {
			dfs(branchName, []string{})
		}
	}

	return cycles
}

// checkMissingParents checks for branches whose parent branches don't exist
func checkMissingParents(eng engine.Engine, allBranches []string) []string {
	var missing []string
	branchSet := make(map[string]bool)
	trunk := eng.Trunk()
	trunkName := trunk.Name

	for _, branch := range allBranches {
		branchSet[branch] = true
	}

	for _, branch := range allBranches {
		if branch == trunkName {
			continue
		}
		branchObj := eng.GetBranch(branch)
		parent := eng.GetParent(branchObj)
		if parent != nil && parent.Name != trunkName {
			if !branchSet[parent.Name] {
				missing = append(missing, branch)
			}
		}
	}

	return missing
}
