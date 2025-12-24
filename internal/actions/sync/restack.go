package sync

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// restackBranches handles restacking branches after sync operations
func restackBranches(ctx *runtime.Context, branchesToRestack []string) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Add current branch stack to restack list
	currentBranch := eng.CurrentBranch()
	if currentBranch != nil {
		if currentBranch.IsTracked() {
			// Get full stack (up to trunk)
			stack := eng.GetFullStack(*currentBranch)
			// Add branches to restack list
			for _, b := range stack {
				branchesToRestack = append(branchesToRestack, b.Name)
			}
		} else if currentBranch.IsTrunk() {
			// If on trunk, restack all branches
			stack := currentBranch.GetRelativeStack(engine.StackRange{RecursiveChildren: true})
			for _, b := range stack {
				branchesToRestack = append(branchesToRestack, b.Name)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueBranches := []engine.Branch{}
	for _, branchName := range branchesToRestack {
		if !seen[branchName] {
			seen[branchName] = true
			uniqueBranches = append(uniqueBranches, eng.GetBranch(branchName))
		}
	}

	// Sort branches topologically (parents before children) for correct restack order
	sortedBranches := eng.SortBranchesTopologically(uniqueBranches)

	// Restack branches
	if len(sortedBranches) > 0 {
		if err := actions.RestackBranches(gctx, sortedBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}
