package actions

import (
	"context"
	"fmt"
	"sync"

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
	c := ctx.Context

	// Pre-calculate which branches should be deleted in parallel
	allTrackedBranches := eng.AllBranchNames()
	type deleteStatus struct {
		shouldDelete bool
		reason       string
	}
	deleteStatuses := make(map[string]deleteStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, branchName := range allTrackedBranches {
		branch := eng.GetBranch(branchName)
		if branch.IsTrunk() {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			shouldDelete, reason := ShouldDeleteBranch(c, name, eng, opts.Force)
			mu.Lock()
			deleteStatuses[name] = deleteStatus{shouldDelete: shouldDelete, reason: reason}
			mu.Unlock()
		}(branchName)
	}
	wg.Wait()

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

		// Use pre-calculated status
		status := deleteStatuses[branchName]
		if status.shouldDelete {
			children := eng.GetChildren(branchName)
			// Add children to process (DFS)
			branchesToProcess = append(branchesToProcess, children...)

			// Mark for deletion with blockers
			blockers := make(map[string]bool)
			for _, child := range children {
				blockers[child] = true
			}
			branchesToDelete[branchName] = blockers

			splog.Debug("Marked %s for deletion. Reason: %s. Blockers: %v", branchName, status.reason, children)
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
				if err := eng.SetParent(c, branchName, newParent); err != nil {
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
		greedilyDeleteUnblockedBranches(c, branchesToDelete, eng, splog)
	}

	return &CleanBranchesResult{
		BranchesWithNewParents: branchesWithNewParents,
	}, nil
}

// greedilyDeleteUnblockedBranches deletes branches that have no blockers
func greedilyDeleteUnblockedBranches(ctx context.Context, branchesToDelete map[string]map[string]bool, eng engine.Engine, splog *tui.Splog) {
	for branchName, blockers := range branchesToDelete {
		if len(blockers) == 0 {
			// No blockers, safe to delete
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}

			// Delete the branch
			if err := eng.DeleteBranch(ctx, branchName); err != nil {
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
			greedilyDeleteUnblockedBranches(ctx, branchesToDelete, eng, splog)
		}
	}
}
