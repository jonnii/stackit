package engine

import (
	"context"
	"fmt"
	"slices"
)

// ApplySplitToCommits creates branches at specified commit points
func (e *engineImpl) ApplySplitToCommits(ctx context.Context, opts ApplySplitOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(opts.BranchNames) != len(opts.BranchPoints) {
		return fmt.Errorf("invalid number of branch names: got %d names but %d branch points", len(opts.BranchNames), len(opts.BranchPoints))
	}

	// Get metadata for the branch being split
	meta, err := e.readMetadataRef(opts.BranchToSplit)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if meta.ParentBranchName == nil {
		return fmt.Errorf("branch %s has no parent", opts.BranchToSplit)
	}

	parentBranchName := *meta.ParentBranchName
	parentRevision := *meta.ParentBranchRevision
	children := e.childrenMap[opts.BranchToSplit]

	// Reverse branch points (newest to oldest -> oldest to newest)
	reversedBranchPoints := make([]int, len(opts.BranchPoints))
	for i, point := range opts.BranchPoints {
		reversedBranchPoints[len(opts.BranchPoints)-1-i] = point
	}

	// Keep track of the last branch's name + SHA for metadata
	lastBranchName := parentBranchName
	lastBranchRevision := parentRevision

	// Create each branch
	for idx, branchName := range opts.BranchNames {
		// Get commit SHA at the offset
		branchRevision, err := e.git.GetCommitSHA(opts.BranchToSplit, reversedBranchPoints[idx])
		if err != nil {
			return fmt.Errorf("failed to get commit SHA at offset %d: %w", reversedBranchPoints[idx], err)
		}

		// Create branch at that SHA
		_, err = e.git.RunGitCommandWithContext(ctx, "branch", "-f", branchName, branchRevision)
		if err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}

		// Preserve PR info if branch name matches original
		var prInfo *PrInfo
		if branchName == opts.BranchToSplit {
			branchToSplit := e.GetBranch(opts.BranchToSplit)
			prInfo, _ = e.GetPrInfo(branchToSplit)
		}

		// Track branch with parent
		newMeta := &Meta{
			ParentBranchName:     &lastBranchName,
			ParentBranchRevision: &lastBranchRevision,
		}

		// Preserve PR info if applicable
		if prInfo != nil {
			newMeta.PrInfo = &PrInfoPersistence{
				Number:  prInfo.Number(),
				Title:   stringPtr(prInfo.Title()),
				Body:    stringPtr(prInfo.Body()),
				IsDraft: boolPtr(prInfo.IsDraft()),
				State:   stringPtr(prInfo.State()),
				Base:    stringPtr(prInfo.Base()),
				URL:     stringPtr(prInfo.URL()),
			}
		}

		if err := e.writeMetadataRef(branchName, newMeta); err != nil {
			return fmt.Errorf("failed to write metadata for %s: %w", branchName, err)
		}

		// Update in-memory cache
		e.parentMap[branchName] = lastBranchName
		e.childrenMap[lastBranchName] = append(e.childrenMap[lastBranchName], branchName)

		// Update last branch info
		lastBranchName = branchName
		lastBranchRevision = branchRevision
	}

	// Update children to point to last branch
	if lastBranchName != opts.BranchToSplit {
		lastBranch := e.GetBranch(lastBranchName)
		for _, childBranchName := range children {
			if err := e.SetParent(ctx, e.GetBranch(childBranchName), lastBranch); err != nil {
				return fmt.Errorf("failed to update parent for %s: %w", childBranchName, err)
			}
		}
	}

	// Delete original branch if not in branchNames
	if !slices.Contains(opts.BranchNames, opts.BranchToSplit) {
		if err := e.DeleteBranch(ctx, e.GetBranch(opts.BranchToSplit)); err != nil {
			return fmt.Errorf("failed to delete original branch: %w", err)
		}
	}

	// Checkout last branch
	e.currentBranch = lastBranchName
	lastBranch := e.GetBranch(lastBranchName)
	if err := e.git.CheckoutBranch(ctx, lastBranch.GetName()); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", lastBranchName, err)
	}

	return nil
}

// Detach detaches HEAD to a specific revision
func (e *engineImpl) Detach(ctx context.Context, revision string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Checkout the revision in detached HEAD state
	_, err := e.git.RunGitCommandWithContext(ctx, "checkout", "--detach", revision)
	if err != nil {
		return fmt.Errorf("failed to detach HEAD: %w", err)
	}

	e.currentBranch = ""
	return nil
}

// DetachAndResetBranchChanges detaches HEAD and soft resets to the parent's merge base,
// leaving the branch's changes as unstaged modifications. This is used by split --by-hunk
// to allow the user to interactively re-stage changes into new branches.
func (e *engineImpl) DetachAndResetBranchChanges(ctx context.Context, branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get branch revision
	branch := e.GetBranch(branchName)
	branchRevision, err := branch.GetRevision()
	if err != nil {
		return fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Get the parent branch
	parentBranchName := e.parentMap[branchName]
	if parentBranchName == "" {
		parentBranchName = e.trunk
	}

	// Get the merge base between this branch and its parent
	mergeBase, err := e.git.GetMergeBase(branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Detach HEAD to the branch revision first
	_, err = e.git.RunGitCommandWithContext(ctx, "checkout", "--detach", branchRevision)
	if err != nil {
		return fmt.Errorf("failed to detach HEAD: %w", err)
	}

	// Soft reset to the merge base - this keeps all the branch's changes
	// but unstages them, allowing the user to re-stage them interactively
	_, err = e.git.RunGitCommandWithContext(ctx, "reset", mergeBase)
	if err != nil {
		return fmt.Errorf("failed to soft reset: %w", err)
	}

	e.currentBranch = ""
	return nil
}

// ForceCheckoutBranch checks out a branch
func (e *engineImpl) ForceCheckoutBranch(ctx context.Context, branch Branch) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	branchName := branch.GetName()
	_, err := e.git.RunGitCommandWithContext(ctx, "checkout", "-f", branchName)
	if err != nil {
		return fmt.Errorf("failed to force checkout branch: %w", err)
	}

	e.currentBranch = branchName
	return nil
}
