package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// CleanBranchesOptions contains options for cleaning branches
type CleanBranchesOptions struct {
	Force bool
}

// CleanBranchesResult contains the result of cleaning branches
type CleanBranchesResult struct {
	BranchesWithNewParents []string
}

// CleanBranches finds and deletes merged/closed branches
// Returns branches whose parents have changed (need restacking)
func CleanBranches(ctx *runtime.Context, opts CleanBranchesOptions) (*CleanBranchesResult, error) {
	eng := ctx.Engine
	splog := ctx.Splog

	// Start from trunk children
	branchesToProcess := eng.GetChildren(eng.Trunk())
	branchesToDelete := make(map[string]map[string]bool) // branch -> set of blocking children
	branchesWithNewParents := []string{}

	// DFS traversal
	for len(branchesToProcess) > 0 {
		// Pop from stack
		branchName := branchesToProcess[len(branchesToProcess)-1]
		branchesToProcess = branchesToProcess[:len(branchesToProcess)-1]

		// Skip if already marked for deletion
		if _, ok := branchesToDelete[branchName]; ok {
			continue
		}

		// Check if should delete
		shouldDelete, _ := shouldDeleteBranch(branchName, eng, opts.Force)
		if shouldDelete {
			children := eng.GetChildren(branchName)
			// Add children to process (DFS)
			branchesToProcess = append(branchesToProcess, children...)

			// Mark for deletion with blockers
			blockers := make(map[string]bool)
			for _, child := range children {
				blockers[child] = true
			}
			branchesToDelete[branchName] = blockers

			splog.Debug("Marked %s for deletion. Blockers: %v", branchName, children)
		} else {
			// Branch is not being deleted
			// If its parent IS being deleted, update parent
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}

			// Find nearest ancestor that isn't being deleted
			newParent := parent
			for {
				if _, isDeleting := branchesToDelete[newParent]; !isDeleting {
					break
				}
				ancestor := eng.GetParent(newParent)
				if ancestor == "" {
					newParent = eng.Trunk()
					break
				}
				newParent = ancestor
			}

			// If parent changed, update it
			if newParent != parent {
				if err := eng.SetParent(branchName, newParent); err != nil {
					return nil, fmt.Errorf("failed to set parent for %s: %w", branchName, err)
				}
				splog.Info("Set parent of %s to %s.",
					tui.ColorBranchName(branchName, false),
					tui.ColorBranchName(newParent, false))
				branchesWithNewParents = append(branchesWithNewParents, branchName)

				// Remove this branch as a blocker for its old parent
				if blockers, ok := branchesToDelete[parent]; ok {
					delete(blockers, branchName)
					branchesToDelete[parent] = blockers
				}
			}
		}

		// Greedily delete unblocked branches
		greedilyDeleteUnblockedBranches(branchesToDelete, eng, splog)
	}

	return &CleanBranchesResult{
		BranchesWithNewParents: branchesWithNewParents,
	}, nil
}

// greedilyDeleteUnblockedBranches deletes branches that have no blockers
func greedilyDeleteUnblockedBranches(branchesToDelete map[string]map[string]bool, eng engine.Engine, splog *tui.Splog) {
	for branchName, blockers := range branchesToDelete {
		if len(blockers) == 0 {
			// No blockers, safe to delete
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}

			// Delete the branch
			if err := eng.DeleteBranch(branchName); err != nil {
				splog.Debug("Failed to delete %s: %v", branchName, err)
				continue
			}

			splog.Info("Deleted branch %s", tui.ColorBranchName(branchName, false))

			// Remove from deletion map
			delete(branchesToDelete, branchName)

			// Remove this branch as a blocker for its parent
			if parentBlockers, ok := branchesToDelete[parent]; ok {
				delete(parentBlockers, branchName)
				branchesToDelete[parent] = parentBlockers
			}

			// Recursively check if parent is now unblocked
			greedilyDeleteUnblockedBranches(branchesToDelete, eng, splog)
		}
	}
}

// shouldDeleteBranch checks if a branch should be deleted
func shouldDeleteBranch(branchName string, eng engine.Engine, force bool) (bool, string) {
	// Check PR info
	prInfo, err := eng.GetPrInfo(branchName)
	if err == nil && prInfo != nil {
		const (
			prStateClosed = "CLOSED"
			prStateMerged = "MERGED"
		)
		if prInfo.State == prStateClosed {
			return true, fmt.Sprintf("%s is closed on GitHub", branchName)
		}
		if prInfo.State == prStateMerged {
			base := prInfo.Base
			if base == "" {
				base = eng.Trunk()
			}
			return true, fmt.Sprintf("%s is merged into %s", branchName, base)
		}
	}

	// Check if merged into trunk
	merged, err := eng.IsMergedIntoTrunk(branchName)
	if err == nil && merged {
		return true, fmt.Sprintf("%s is merged into %s", branchName, eng.Trunk())
	}

	// Check if empty
	empty, err := eng.IsBranchEmpty(branchName)
	if err == nil && empty {
		// Only delete empty branches if they have a PR
		if prInfo != nil && prInfo.Number != nil {
			return true, fmt.Sprintf("%s is empty", branchName)
		}
	}

	// If force, don't prompt
	if force {
		return false, ""
	}

	// For now, we don't prompt interactively
	// In a full implementation, we would prompt here
	return false, ""
}
