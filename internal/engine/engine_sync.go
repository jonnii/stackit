package engine

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/git"
)

// PullTrunk pulls the trunk branch from remote
func (e *engineImpl) PullTrunk(ctx context.Context) (PullResult, error) {
	remote := git.GetRemote()
	e.mu.RLock()
	trunk := e.trunk
	e.mu.RUnlock()
	gitResult, err := git.PullBranch(ctx, remote, trunk)
	if err != nil {
		return PullConflict, err
	}

	// Convert git.PullResult to engine.PullResult
	var result PullResult
	switch gitResult {
	case git.PullDone:
		result = PullDone
	case git.PullUnneeded:
		result = PullUnneeded
	case git.PullConflict:
		result = PullConflict
	default:
		result = PullConflict
	}

	// Rebuild to refresh branch cache
	if err := e.rebuild(); err != nil {
		return result, fmt.Errorf("failed to rebuild after pull: %w", err)
	}

	return result, nil
}

// ResetTrunkToRemote resets trunk to match remote
func (e *engineImpl) ResetTrunkToRemote(ctx context.Context) error {
	remote := git.GetRemote()

	e.mu.RLock()
	trunk := e.trunk
	currentBranch := e.currentBranch
	e.mu.RUnlock()

	// Get remote SHA
	remoteSha, err := git.GetRemoteSha(remote, trunk)
	if err != nil {
		return fmt.Errorf("failed to get remote SHA: %w", err)
	}

	// Checkout trunk
	trunkBranch := e.Trunk()
	if err := git.CheckoutBranch(ctx, trunkBranch.Name); err != nil {
		return fmt.Errorf("failed to checkout trunk: %w", err)
	}

	// Hard reset to remote
	if err := git.HardReset(ctx, remoteSha); err != nil {
		// Try to switch back
		if currentBranch != "" {
			currentBranchObj := e.GetBranch(currentBranch)
			_ = git.CheckoutBranch(ctx, currentBranchObj.Name)
		}
		return fmt.Errorf("failed to reset trunk: %w", err)
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != trunk {
		currentBranchObj := e.GetBranch(currentBranch)
		if err := git.CheckoutBranch(ctx, currentBranchObj.Name); err != nil {
			return fmt.Errorf("failed to switch back: %w", err)
		}
	}

	// Rebuild to refresh branch cache
	if err := e.rebuild(); err != nil {
		return fmt.Errorf("failed to rebuild after reset: %w", err)
	}

	return nil
}

// RestackBranch rebases a branch onto its parent
// If the parent has been merged/deleted, it will automatically reparent to the nearest valid ancestor
func (e *engineImpl) RestackBranch(ctx context.Context, branch Branch, rebuildAfterRestack bool) (RestackBranchResult, error) {
	branchName := branch.Name
	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		// RESILIENCY: Try to auto-discover parent if branch is not tracked
		ancestors, err := e.FindMostRecentTrackedAncestors(ctx, branchName)
		if err == nil && len(ancestors) > 0 {
			parent = ancestors[0]
			// Auto-track the branch
			if err := e.TrackBranch(ctx, branchName, parent); err == nil {
				ok = true
			}
		}

		if !ok {
			return RestackBranchResult{Result: RestackUnneeded}, fmt.Errorf("branch %s is not tracked", branchName)
		}
	}

	// Track reparenting info
	var reparented bool
	var oldParent string

	// Check if parent needs reparenting (merged, deleted, or has MERGED PR state)
	e.mu.RLock()
	needsReparent := e.shouldReparentBranch(ctx, parent)
	e.mu.RUnlock()

	if needsReparent {
		oldParent = parent

		// Find nearest valid ancestor
		e.mu.RLock()
		newParent := e.findNearestValidAncestor(ctx, branchName)
		e.mu.RUnlock()

		// Reparent to the nearest valid ancestor
		if err := e.SetParent(ctx, branchName, newParent); err != nil {
			return RestackBranchResult{Result: RestackConflict}, fmt.Errorf("failed to reparent %s to %s: %w", branchName, newParent, err)
		}
		parent = newParent
		reparented = true
	}

	// Get parent revision (needed for rebasedBranchBase even if restack is unneeded)
	parentBranch := e.GetBranch(parent)
	parentRev, err := parentBranch.GetRevision()
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, fmt.Errorf("failed to get parent revision: %w", err)
	}

	// Get metadata (read once to avoid duplicate disk I/O)
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Check if branch needs restacking using cached metadata
	if meta.ParentBranchRevision != nil && *meta.ParentBranchRevision == parentRev {
		return RestackBranchResult{
			Result:            RestackUnneeded,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, nil
	}

	oldParentRev := parentRev
	if meta.ParentBranchRevision != nil {
		oldParentRev = *meta.ParentBranchRevision
	}

	// If parent hasn't changed, no need to restack (early exit before expensive operations)
	if parentRev == oldParentRev {
		return RestackBranchResult{
			Result:            RestackUnneeded,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, nil
	}

	// RESILIENCY: If oldParentRev is no longer an ancestor of branchName,
	// or if it's empty, find the actual merge base. This handles cases where
	// the parent was amended or rebased outside of stackit.
	if oldParentRev != "" {
		if isAncestor, _ := git.IsAncestor(oldParentRev, branchName); !isAncestor {
			if mergeBase, err := git.GetMergeBase(branchName, parent); err == nil {
				oldParentRev = mergeBase
			}
		}
	} else {
		// No old parent revision in metadata, try to find merge base
		if mergeBase, err := git.GetMergeBase(branchName, parent); err == nil {
			oldParentRev = mergeBase
		}
	}

	// Check again after resiliency logic - parent might still be unchanged
	if parentRev == oldParentRev {
		return RestackBranchResult{
			Result:            RestackUnneeded,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, nil
	}

	// Perform rebase
	gitResult, err := git.Rebase(ctx, branchName, parent, oldParentRev)
	if err != nil {
		return RestackBranchResult{
			Result:            RestackConflict,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, err
	}

	if gitResult == git.RebaseConflict {
		return RestackBranchResult{
			Result:            RestackConflict,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, nil
	}

	// Update metadata with new parent revision
	meta.ParentBranchRevision = &parentRev
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return RestackBranchResult{
			Result:            RestackDone,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, fmt.Errorf("failed to update metadata: %w", err)
	}

	// Update cache incrementally if requested (much faster than full rebuild)
	if rebuildAfterRestack {
		e.updateBranchInCache(branchName)
	}

	return RestackBranchResult{
		Result:            RestackDone,
		RebasedBranchBase: parentRev,
		Reparented:        reparented,
		OldParent:         oldParent,
		NewParent:         parent,
	}, nil
}

// RestackBranches restacks multiple branches using batch I/O operations for performance
func (e *engineImpl) RestackBranches(ctx context.Context, branches []Branch) (RestackBatchResult, error) {
	results := make(map[string]RestackBranchResult)

	// Filter out trunk branches
	var branchesToRestack []Branch
	for _, branch := range branches {
		if !branch.IsTrunk() {
			branchesToRestack = append(branchesToRestack, branch)
		} else {
			results[branch.Name] = RestackBranchResult{Result: RestackUnneeded}
		}
	}

	if len(branchesToRestack) == 0 {
		return RestackBatchResult{Results: results}, nil
	}

	// Batch preparation phase - collect all data upfront for performance

	// 1. Collect branch names for batch operations
	branchNames := make([]string, len(branchesToRestack))
	for i, branch := range branchesToRestack {
		branchNames[i] = branch.Name
	}

	// 2. Batch read all metadata using the new batch method
	metadataMap, err := git.BatchReadMetadataRefs(branchNames)
	if err != nil {
		return RestackBatchResult{}, fmt.Errorf("failed to batch read metadata: %w", err)
	}

	// 3. Collect unique parent branches for batch operations
	parentSet := make(map[string]bool)
	for _, branch := range branchesToRestack {
		e.mu.RLock()
		if parent, ok := e.parentMap[branch.Name]; ok {
			parentSet[parent] = true
		}
		e.mu.RUnlock()
	}
	uniqueParents := make([]string, 0, len(parentSet))
	for parent := range parentSet {
		uniqueParents = append(uniqueParents, parent)
	}

	// 4. Batch get parent revisions using the new batch method
	parentRevisions, _ := git.BatchGetRevisions(uniqueParents)

	// Processing phase - restack each branch using pre-fetched data
	needsCacheRebuild := false

	for i, branch := range branchesToRestack {
		branchName := branch.Name

		// Get pre-fetched metadata instead of reading from disk
		meta := metadataMap[branchName]
		if meta == nil {
			meta = &git.Meta{} // Empty meta if not found
		}

		// Get cached parent info
		e.mu.RLock()
		parent, parentExists := e.parentMap[branchName]
		e.mu.RUnlock()

		if !parentExists {
			// Try auto-discovery like individual RestackBranch does
			ancestors, err := e.FindMostRecentTrackedAncestors(ctx, branchName)
			if err == nil && len(ancestors) > 0 {
				parent = ancestors[0]
				// Auto-track the branch
				if err := e.TrackBranch(ctx, branchName, parent); err == nil {
					parentExists = true
					// Add to unique parents if not already there
					if _, exists := parentSet[parent]; !exists {
						uniqueParents = append(uniqueParents, parent)
						parentSet[parent] = true
						// Get revision for new parent
						if rev, err := git.GetRevision(parent); err == nil {
							parentRevisions[parent] = rev
						}
					}
				}
			}
		}

		if !parentExists {
			results[branchName] = RestackBranchResult{Result: RestackUnneeded}
			continue
		}

		// Check if parent needs reparenting (do this per-branch to detect merges during batch)
		var reparented bool
		var oldParent string
		e.mu.RLock()
		needsReparent := e.shouldReparentBranch(ctx, parent)
		e.mu.RUnlock()
		if needsReparent {
			oldParent = parent
			reparented = true

			// Find nearest valid ancestor
			e.mu.RLock()
			newParent := e.findNearestValidAncestor(ctx, branchName)
			e.mu.RUnlock()

			// Ensure we have the revision for the new parent
			if _, hasRev := parentRevisions[newParent]; !hasRev {
				if rev, err := git.GetRevision(newParent); err == nil {
					parentRevisions[newParent] = rev
				}
			}

			// Reparent to the nearest valid ancestor
			if err := e.SetParent(ctx, branchName, newParent); err != nil {
				// Convert remaining branches to []string for RestackBatchResult
				remainingBranchNames := make([]string, len(branchesToRestack[i+1:]))
				for j, b := range branchesToRestack[i+1:] {
					remainingBranchNames[j] = b.Name
				}
				return RestackBatchResult{
					ConflictBranch:    branchName,
					RebasedBranchBase: parentRevisions[newParent],
					RemainingBranches: remainingBranchNames,
					Results:           results,
				}, fmt.Errorf("failed to reparent %s to %s: %w", branchName, newParent, err)
			}
			parent = newParent
		}

		// Get parent revision from pre-fetched cache instead of individual lookup
		parentRev, hasParentRev := parentRevisions[parent]
		if !hasParentRev {
			// Fallback to individual lookup if batch failed
			var err error
			parentRev, err = git.GetRevision(parent)
			if err != nil {
				results[branchName] = RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}
				// Convert remaining branches to []string for RestackBatchResult
				remainingBranchNames := make([]string, len(branchesToRestack[i+1:]))
				for j, b := range branchesToRestack[i+1:] {
					remainingBranchNames[j] = b.Name
				}
				return RestackBatchResult{
					ConflictBranch:    branchName,
					RebasedBranchBase: parentRev,
					RemainingBranches: remainingBranchNames,
					Results:           results,
				}, fmt.Errorf("failed to get parent revision for %s: %w", parent, err)
			}
		}

		// Check if branch needs restacking using pre-fetched metadata
		oldParentRev := parentRev
		if meta.ParentBranchRevision != nil {
			oldParentRev = *meta.ParentBranchRevision
		}

		// Early exit if parent hasn't changed
		if parentRev == oldParentRev {
			results[branchName] = RestackBranchResult{
				Result:            RestackUnneeded,
				RebasedBranchBase: parentRev,
				Reparented:        reparented,
				OldParent:         oldParent,
				NewParent:         parent,
			}
			continue
		}

		// Perform resiliency checks (IsAncestor, GetMergeBase) only when needed
		if oldParentRev != "" {
			if isAncestor, _ := git.IsAncestor(oldParentRev, branchName); !isAncestor {
				if mergeBase, err := git.GetMergeBase(branchName, parent); err == nil {
					oldParentRev = mergeBase
				}
			}
		} else {
			// No old parent revision in metadata, try to find merge base
			if mergeBase, err := git.GetMergeBase(branchName, parent); err == nil {
				oldParentRev = mergeBase
			}
		}

		// Final check after resiliency logic
		if parentRev == oldParentRev {
			results[branchName] = RestackBranchResult{
				Result:            RestackUnneeded,
				RebasedBranchBase: parentRev,
				Reparented:        reparented,
				OldParent:         oldParent,
				NewParent:         parent,
			}
			continue
		}

		// Perform rebase
		gitResult, err := git.Rebase(ctx, branchName, parent, oldParentRev)
		if err != nil {
			results[branchName] = RestackBranchResult{
				Result:            RestackConflict,
				RebasedBranchBase: parentRev,
				Reparented:        reparented,
				OldParent:         oldParent,
				NewParent:         parent,
			}
			// Convert remaining branches to []string for RestackBatchResult
			remainingBranchNames := make([]string, len(branchesToRestack[i+1:]))
			for j, b := range branchesToRestack[i+1:] {
				remainingBranchNames[j] = b.Name
			}
			return RestackBatchResult{
				ConflictBranch:    branchName,
				RebasedBranchBase: parentRev,
				RemainingBranches: remainingBranchNames,
				Results:           results,
			}, err
		}

		if gitResult == git.RebaseConflict {
			results[branchName] = RestackBranchResult{
				Result:            RestackConflict,
				RebasedBranchBase: parentRev,
				Reparented:        reparented,
				OldParent:         oldParent,
				NewParent:         parent,
			}
			// Convert remaining branches to []string for RestackBatchResult
			remainingBranchNames := make([]string, len(branchesToRestack[i+1:]))
			for j, b := range branchesToRestack[i+1:] {
				remainingBranchNames[j] = b.Name
			}
			return RestackBatchResult{
				ConflictBranch:    branchName,
				RebasedBranchBase: parentRev,
				RemainingBranches: remainingBranchNames,
				Results:           results,
			}, nil
		}

		// Update metadata (SetParent already updated parent name, just need to update revision)
		meta.ParentBranchRevision = &parentRev
		if err := git.WriteMetadataRef(branchName, meta); err != nil {
			results[branchName] = RestackBranchResult{
				Result:            RestackDone,
				RebasedBranchBase: parentRev,
				Reparented:        reparented,
				OldParent:         oldParent,
				NewParent:         parent,
			}
			// Convert remaining branches to []string for RestackBatchResult
			remainingBranchNames := make([]string, len(branchesToRestack[i+1:]))
			for j, b := range branchesToRestack[i+1:] {
				remainingBranchNames[j] = b.Name
			}
			return RestackBatchResult{
				ConflictBranch:    branchName,
				RebasedBranchBase: parentRev,
				RemainingBranches: remainingBranchNames,
				Results:           results,
			}, fmt.Errorf("failed to write metadata for %s: %w", branchName, err)
		}

		results[branchName] = RestackBranchResult{
			Result:            RestackDone,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}
		needsCacheRebuild = true
	}

	// Single cache rebuild at the end
	if needsCacheRebuild {
		if err := e.rebuild(); err != nil {
			return RestackBatchResult{
				Results: results,
			}, fmt.Errorf("failed to rebuild after batch restack: %w", err)
		}
	}

	return RestackBatchResult{
		Results: results,
	}, nil
}

// ContinueRebase continues an in-progress rebase
func (e *engineImpl) ContinueRebase(ctx context.Context, rebasedBranchBase string) (ContinueRebaseResult, error) {
	// Call git rebase --continue
	result, err := git.RebaseContinue(ctx)
	if err != nil {
		return ContinueRebaseResult{Result: int(git.RebaseConflict)}, err
	}

	if result == git.RebaseConflict {
		return ContinueRebaseResult{Result: int(git.RebaseConflict)}, nil
	}

	// Get current branch after successful rebase
	branchName, err := git.GetCurrentBranch()
	if err != nil {
		return ContinueRebaseResult{}, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Update metadata for the rebased branch
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return ContinueRebaseResult{}, fmt.Errorf("failed to read metadata: %w", err)
	}

	meta.ParentBranchRevision = &rebasedBranchBase
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return ContinueRebaseResult{}, fmt.Errorf("failed to update metadata: %w", err)
	}

	// Rebuild to refresh cache
	if err := e.rebuild(); err != nil {
		return ContinueRebaseResult{}, fmt.Errorf("failed to rebuild after continue: %w", err)
	}

	return ContinueRebaseResult{
		Result:     int(git.RebaseDone),
		BranchName: branchName,
	}, nil
}
