package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"

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

	// Load metadata for each branch in parallel
	type branchMeta struct {
		name string
		meta *git.Meta
	}
	metaChan := make(chan branchMeta, len(branches))
	var wg sync.WaitGroup

	for _, branchName := range branches {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			meta, err := git.ReadMetadataRef(name)
			if err != nil {
				return
			}
			metaChan <- branchMeta{name: name, meta: meta}
		}(branchName)
	}

	// Close channel when all workers are done
	go func() {
		wg.Wait()
		close(metaChan)
	}()

	// Collect results and populate maps sequentially to avoid lock contention/races
	for bm := range metaChan {
		if bm.meta.ParentBranchName != nil {
			parent := *bm.meta.ParentBranchName
			e.parentMap[bm.name] = parent
			e.childrenMap[parent] = append(e.childrenMap[parent], bm.name)
		}
	}

	// Sort children by name for deterministic traversal
	for _, children := range e.childrenMap {
		sort.Strings(children)
	}

	return nil
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
	prInfo, err := e.GetPrInfo(ctx, parentBranchName)
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
func (e *engineImpl) getRelativeStackUpstackInternal(branchName string) []string {
	result := []string{}
	visited := make(map[string]bool)

	var collectDescendants func(string)
	collectDescendants = func(branch string) {
		if visited[branch] {
			return
		}
		visited[branch] = true

		// Don't include the starting branch
		if branch != branchName {
			result = append(result, branch)
		}

		children := e.childrenMap[branch]
		for _, child := range children {
			collectDescendants(child)
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
