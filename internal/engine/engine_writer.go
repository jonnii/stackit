package engine

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/git"
)

// PushBranch pushes a branch to the remote
func (e *engineImpl) PushBranch(ctx context.Context, branchName string, remote string, force bool, forceWithLease bool) error {
	return git.PushBranch(ctx, branchName, remote, force, forceWithLease)
}

// TrackBranch tracks a branch with a parent branch
func (e *engineImpl) TrackBranch(ctx context.Context, branchName string, parentBranchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update current branch if it changed
	if current, err := git.GetCurrentBranch(); err == nil {
		e.currentBranch = current
	}

	// Validate branch exists
	branchExists := false
	for _, name := range e.branches {
		if name == branchName {
			branchExists = true
			break
		}
	}
	if !branchExists {
		// Refresh branches list
		branches, err := git.GetAllBranchNames()
		if err != nil {
			return fmt.Errorf("failed to get branches: %w", err)
		}
		e.branches = branches
		branchExists = false
		for _, name := range e.branches {
			if name == branchName {
				branchExists = true
				break
			}
		}
		if !branchExists {
			return fmt.Errorf("branch %s does not exist", branchName)
		}
	}

	// Validate parent exists (or is trunk)
	if parentBranchName != e.trunk {
		parentExists := false
		for _, name := range e.branches {
			if name == parentBranchName {
				parentExists = true
				break
			}
		}
		if !parentExists {
			// Refresh branches list to check again
			branches, err := git.GetAllBranchNames()
			if err != nil {
				return fmt.Errorf("failed to get branches: %w", err)
			}
			e.branches = branches
			parentExists = false
			for _, name := range e.branches {
				if name == parentBranchName {
					parentExists = true
					break
				}
			}
			if !parentExists {
				return fmt.Errorf("parent branch %s does not exist", parentBranchName)
			}
		}
	}

	return e.setParentInternal(ctx, branchName, parentBranchName)
}

// UntrackBranch stops tracking a branch by deleting its metadata
func (e *engineImpl) UntrackBranch(branchName string) error {
	if e.IsTrunkInternal(branchName) {
		return fmt.Errorf("cannot untrack trunk branch")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Delete metadata
	if err := git.DeleteMetadataRef(branchName); err != nil {
		return fmt.Errorf("failed to delete metadata ref: %w", err)
	}

	// Rebuild cache (already holding lock, so call rebuildInternal)
	return e.rebuildInternal(true)
}

// DeleteBranch deletes a branch and its metadata
func (e *engineImpl) DeleteBranch(ctx context.Context, branchName string) error {
	if e.IsTrunkInternal(branchName) {
		return fmt.Errorf("cannot delete trunk branch")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Get children before deletion
	children := make([]string, len(e.childrenMap[branchName]))
	copy(children, e.childrenMap[branchName])

	// Get parent
	parent, ok := e.parentMap[branchName]
	if !ok {
		parent = e.trunk
	}

	// If deleting current branch, switch to trunk first
	if branchName == e.currentBranch {
		// Access trunk directly while holding the lock (avoid deadlock from e.Trunk() trying to acquire RLock)
		trunkBranch := Branch{Name: e.trunk, Reader: e}
		if err := git.CheckoutBranch(ctx, trunkBranch.Name); err != nil {
			return fmt.Errorf("failed to switch to trunk before deleting current branch: %w", err)
		}
		e.currentBranch = e.trunk
	}

	// Delete git branch
	branch := e.GetBranch(branchName)
	if err := git.DeleteBranch(ctx, branch.Name); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// Delete metadata
	_ = git.DeleteMetadataRef(branchName)

	// Update children to point to parent
	for _, child := range children {
		if err := e.setParentInternal(ctx, child, parent); err != nil {
			continue
		}
	}

	// Remove from parent's children list
	if parent != "" {
		parentChildren := e.childrenMap[parent]
		for i, c := range parentChildren {
			if c == branchName {
				e.childrenMap[parent] = append(parentChildren[:i], parentChildren[i+1:]...)
				break
			}
		}
	}

	// Remove from maps
	delete(e.parentMap, branchName)
	delete(e.childrenMap, branchName)

	// Remove from branches list
	for i, b := range e.branches {
		if b == branchName {
			e.branches = append(e.branches[:i], e.branches[i+1:]...)
			break
		}
	}

	return nil
}

// DeleteBranches deletes multiple branches and returns the children that need restacking
func (e *engineImpl) DeleteBranches(ctx context.Context, branchNames []string) ([]string, error) {
	// Identify all children of all branches to be deleted
	allChildren := make(map[string]bool)
	toDeleteSet := make(map[string]bool)
	for _, b := range branchNames {
		toDeleteSet[b] = true
		children := e.GetChildrenInternal(b)
		for _, child := range children {
			allChildren[child.Name] = true
		}
	}

	// Remove branches that are also being deleted from the children set
	for _, b := range branchNames {
		delete(allChildren, b)
	}

	// Delete branches
	for _, b := range branchNames {
		if err := e.DeleteBranch(ctx, b); err != nil {
			return nil, fmt.Errorf("failed to delete branch %s: %w", b, err)
		}
	}

	// Convert children map to slice
	childrenToRestack := make([]string, 0, len(allChildren))
	for child := range allChildren {
		childrenToRestack = append(childrenToRestack, child)
	}

	return childrenToRestack, nil
}

// SetParent updates a branch's parent
func (e *engineImpl) SetParent(ctx context.Context, branchName string, parentBranchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.setParentInternal(ctx, branchName, parentBranchName)
}

// SetScope updates a branch's scope
func (e *engineImpl) SetScope(branch Branch, scope Scope) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	branchName := branch.Name

	// Read existing metadata
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Update scope
	if scope.IsEmpty() {
		meta.Scope = nil
	} else {
		scopeStr := scope.String()
		meta.Scope = &scopeStr
	}

	// Write metadata
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Update in-memory map
	if scope.IsEmpty() {
		delete(e.scopeMap, branchName)
	} else {
		e.scopeMap[branchName] = scope.String()
	}

	return nil
}

// RenameBranch renames a branch and its metadata
func (e *engineImpl) RenameBranch(ctx context.Context, oldBranch, newBranch Branch) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	oldName := oldBranch.Name
	newName := newBranch.Name

	// Rename git branch
	if err := git.RenameBranch(ctx, oldName, newName); err != nil {
		return err
	}

	// Rename metadata ref
	if err := git.RenameMetadataRef(oldName, newName); err != nil {
		// Attempt to undo git rename
		_ = git.RenameBranch(ctx, newName, oldName)
		return err
	}

	// Rebuild in-memory state to be safe
	return e.rebuildInternal(true)
}

// setParentInternal updates parent without locking (caller must hold lock)
func (e *engineImpl) setParentInternal(ctx context.Context, branchName string, parentBranchName string) error {
	// Get new parent revision
	parentRev, err := git.GetMergeBase(branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Read existing metadata
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Update parent
	oldParent := ""
	if meta.ParentBranchName != nil {
		oldParent = *meta.ParentBranchName
	}

	// Only update ParentBranchRevision if it's currently nil, invalid, or if we're not
	// in a "parent merged into trunk" situation.
	shouldUpdateRevision := true
	if oldParent != "" && oldParent != parentBranchName && meta.ParentBranchRevision != nil && *meta.ParentBranchRevision != "" {
		// Check if existing revision is still a valid ancestor of the branch
		if isAncestor, _ := git.IsAncestor(*meta.ParentBranchRevision, branchName); isAncestor {
			// Check if the old parent was merged into the new parent (the "merge" case)
			// OR if the new parent is the same as the old parent (no change)
			// We use the branch name to check for merging.
			if merged, _ := git.IsMerged(ctx, oldParent, parentBranchName); merged {
				shouldUpdateRevision = false
			}
		}
	}

	meta.ParentBranchName = &parentBranchName
	if shouldUpdateRevision {
		meta.ParentBranchRevision = &parentRev
	}

	// Write metadata
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Update in-memory maps
	if oldParent != "" {
		// Remove from old parent's children
		oldChildren := e.childrenMap[oldParent]
		for i, c := range oldChildren {
			if c == branchName {
				e.childrenMap[oldParent] = append(oldChildren[:i], oldChildren[i+1:]...)
				break
			}
		}
	}

	// Add to new parent's children
	e.parentMap[branchName] = parentBranchName
	if e.childrenMap[parentBranchName] == nil {
		e.childrenMap[parentBranchName] = []string{}
	}

	// Check if already in children list
	found := false
	for _, c := range e.childrenMap[parentBranchName] {
		if c == branchName {
			found = true
			break
		}
	}
	if !found {
		e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)
	}

	return nil
}
