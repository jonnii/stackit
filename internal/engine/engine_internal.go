package engine

import (
	"context"
	"fmt"
	"sort"

	"stackit.dev/stackit/internal/git"
)

// rebuildInternal is the internal rebuild logic without locking
// refreshCurrentBranch indicates whether to refresh currentBranch from Git
func (e *engineImpl) rebuildInternal(refreshCurrentBranch bool) error {
	// Get all branch names
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}
	e.branches = branches

	// Refresh current branch from Git if requested (needed when called from Rebuild/Reset after branch switches)
	if refreshCurrentBranch {
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			// Not on a branch (e.g., detached HEAD) - that's okay
			e.currentBranch = ""
		} else {
			e.currentBranch = currentBranch
		}
	}

	// Reset maps
	e.parentMap = make(map[string]string)
	e.childrenMap = make(map[string][]string)
	e.scopeMap = make(map[string]string)

	// Load metadata for each branch in parallel
	allMeta, _ := git.BatchReadMetadataRefs(branches)

	// Collect results and populate maps sequentially to avoid lock contention/races
	for name, meta := range allMeta {
		if meta.ParentBranchName != nil {
			parent := *meta.ParentBranchName
			e.parentMap[name] = parent
			e.childrenMap[parent] = append(e.childrenMap[parent], name)
		}
		if meta.Scope != nil {
			e.scopeMap[name] = *meta.Scope
		}
	}

	// Sort children by name for deterministic traversal
	for _, children := range e.childrenMap {
		sort.Strings(children)
	}

	return nil
}

// updateBranchInCache updates the cache for a specific branch after restack/metadata changes
func (e *engineImpl) updateBranchInCache(branchName string) {
	// Read metadata for this branch
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		// If metadata doesn't exist, remove branch from all maps
		if oldParent, exists := e.parentMap[branchName]; exists {
			delete(e.parentMap, branchName)
			// Remove from old parent's children list
			if children, ok := e.childrenMap[oldParent]; ok {
				for i, child := range children {
					if child == branchName {
						e.childrenMap[oldParent] = append(children[:i], children[i+1:]...)
						break
					}
				}
				// Remove empty children lists
				if len(e.childrenMap[oldParent]) == 0 {
					delete(e.childrenMap, oldParent)
				}
			}
		}
		delete(e.scopeMap, branchName)
	}

	// Get the old parent before updating
	oldParent := e.parentMap[branchName]

	// Update parent map
	if meta.ParentBranchName != nil {
		e.parentMap[branchName] = *meta.ParentBranchName
	} else {
		delete(e.parentMap, branchName)
	}

	// Update scope map
	if meta.Scope != nil {
		e.scopeMap[branchName] = *meta.Scope
	} else {
		delete(e.scopeMap, branchName)
	}

	// Update children map - remove from old parent, add to new parent
	newParent := ""
	if meta.ParentBranchName != nil {
		newParent = *meta.ParentBranchName
	}

	// Remove from old parent's children if parent changed
	if oldParent != "" && oldParent != newParent {
		if children, ok := e.childrenMap[oldParent]; ok {
			for i, child := range children {
				if child == branchName {
					e.childrenMap[oldParent] = append(children[:i], children[i+1:]...)
					break
				}
			}
			// Remove empty children lists
			if len(e.childrenMap[oldParent]) == 0 {
				delete(e.childrenMap, oldParent)
			}
		}
	}

	// Add to new parent's children if it has a parent
	if newParent != "" {
		e.childrenMap[newParent] = append(e.childrenMap[newParent], branchName)
		// Sort for deterministic traversal
		sort.Strings(e.childrenMap[newParent])
	}
}

// rebuild loads all branches and their metadata from Git
func (e *engineImpl) rebuild() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Don't refresh currentBranch here - it should already be set correctly
	return e.rebuildInternal(false)
}

// shouldReparentBranch checks if a parent branch should be reparented
// Returns true if the parent branch:
// - No longer exists locally
// - Has been merged into trunk
// - Has a "MERGED" PR state in metadata
func (e *engineImpl) shouldReparentBranch(ctx context.Context, parentBranchName string) bool {
	// Check if parent is trunk (no need to reparent)
	if parentBranchName == e.trunk {
		return false
	}

	// Check if parent branch still exists locally
	parentExists := false
	for _, name := range e.branches {
		if name == parentBranchName {
			parentExists = true
			break
		}
	}
	if !parentExists {
		return true
	}

	// Check if parent has been merged into trunk
	merged, err := git.IsMerged(ctx, parentBranchName, e.trunk)
	if err == nil && merged {
		return true
	}

	// Check if parent has "MERGED" PR state in metadata
	prInfo, err := e.GetPrInfo(parentBranchName)
	if err == nil && prInfo != nil && prInfo.State == "MERGED" {
		return true
	}

	return false
}

// findNearestValidAncestor finds the nearest ancestor that hasn't been merged/deleted
// Returns trunk if all ancestors have been merged
func (e *engineImpl) findNearestValidAncestor(ctx context.Context, branchName string) string {
	current := e.parentMap[branchName]

	for current != "" && current != e.trunk {
		if !e.shouldReparentBranch(ctx, current) {
			return current
		}
		// Move to the next parent
		parent, ok := e.parentMap[current]
		if !ok {
			break
		}
		current = parent
	}

	return e.trunk
}

// getRelativeStackUpstackInternal is the internal implementation without lock
func (e *engineImpl) getRelativeStackUpstackInternal(branchName string) []Branch {
	result := []Branch{}
	visited := make(map[string]bool)

	var collectDescendants func(string)
	collectDescendants = func(branch string) {
		if visited[branch] {
			return
		}
		visited[branch] = true

		// Don't include the starting branch
		if branch != branchName {
			result = append(result, Branch{Name: branch, Reader: e})
		}

		children := e.GetChildrenInternal(branch)
		for _, child := range children {
			collectDescendants(child.Name)
		}
	}

	collectDescendants(branchName)
	return result
}

// Helper functions
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
