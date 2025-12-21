package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
)

// engineImpl is a minimal implementation of the Engine interface
type engineImpl struct {
	repoRoot      string
	trunk         string
	currentBranch string
	branches      []string
	parentMap     map[string]string   // branch -> parent
	childrenMap   map[string][]string // branch -> children
	remoteShas    map[string]string   // branch -> remote SHA (populated by PopulateRemoteShas)
	mu            sync.RWMutex
}

// NewEngine creates a new engine instance
func NewEngine(repoRoot string) (Engine, error) {
	// Initialize git repository
	if err := git.InitDefaultRepo(); err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	e := &engineImpl{
		repoRoot:    repoRoot,
		parentMap:   make(map[string]string),
		childrenMap: make(map[string][]string),
		remoteShas:  make(map[string]string),
	}

	// Get trunk from config
	trunk, err := config.GetTrunk(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get trunk: %w", err)
	}
	e.trunk = trunk

	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		// Not on a branch - that's okay
		currentBranch = ""
	}
	e.currentBranch = currentBranch

	// Load branches and metadata
	// Don't refresh currentBranch here since we just set it
	if err := e.rebuildInternal(false); err != nil {
		return nil, fmt.Errorf("failed to rebuild engine: %w", err)
	}

	return e, nil
}

// rebuild loads all branches and their metadata from Git
func (e *engineImpl) rebuild() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Don't refresh currentBranch here - it should already be set correctly
	return e.rebuildInternal(false)
}

// AllBranchNames returns all branch names
func (e *engineImpl) AllBranchNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.branches
}

// CurrentBranch returns the current branch name
func (e *engineImpl) CurrentBranch() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentBranch
}

// Trunk returns the trunk branch name
func (e *engineImpl) Trunk() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.trunk
}

// GetParent returns the parent branch name (empty string if no parent)
func (e *engineImpl) GetParent(branchName string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if parent, ok := e.parentMap[branchName]; ok {
		return parent
	}
	return ""
}

// GetChildren returns the children branches
func (e *engineImpl) GetChildren(branchName string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if children, ok := e.childrenMap[branchName]; ok {
		return children
	}
	return []string{}
}

// GetRelativeStack returns the stack relative to a branch
// Returns branches in order: ancestors (if RecursiveParents), current (if IncludeCurrent), descendants (if RecursiveChildren)
func (e *engineImpl) GetRelativeStack(branchName string, scope Scope) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []string{}

	// Add ancestors if RecursiveParents is true (excluding trunk)
	if scope.RecursiveParents {
		current := branchName
		ancestors := []string{}
		for {
			if current == e.trunk {
				break
			}
			parent, ok := e.parentMap[current]
			if !ok || parent == e.trunk {
				break
			}
			ancestors = append([]string{parent}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch if IncludeCurrent is true
	if scope.IncludeCurrent {
		result = append(result, branchName)
	}

	// Add descendants if RecursiveChildren is true
	if scope.RecursiveChildren {
		descendants := e.getRelativeStackUpstackInternal(branchName)
		result = append(result, descendants...)
	}

	return result
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

// IsTrunk checks if a branch is the trunk
func (e *engineImpl) IsTrunk(branchName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return branchName == e.trunk
}

// IsBranchTracked checks if a branch is tracked (has metadata)
func (e *engineImpl) IsBranchTracked(branchName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.parentMap[branchName]
	return ok
}

// IsBranchFixed checks if a branch needs restacking
// A branch is fixed if its parent revision matches the stored parent revision
func (e *engineImpl) IsBranchFixed(ctx context.Context, branchName string) bool {
	if e.IsTrunk(branchName) {
		return true
	}

	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		return true // Not tracked, consider it fixed
	}

	// Get current parent revision
	parentRev, err := git.GetRevision(ctx, parent)
	if err != nil {
		return false // Can't determine, assume needs restack
	}

	// Get stored parent revision from metadata
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return false // No metadata, assume needs restack
	}

	if meta.ParentBranchRevision == nil {
		return false // No stored revision, needs restack
	}

	// Branch is fixed if stored revision matches current parent revision
	return *meta.ParentBranchRevision == parentRev
}

// GetCommitDate returns the commit date for a branch
func (e *engineImpl) GetCommitDate(ctx context.Context, branchName string) (time.Time, error) {
	return git.GetCommitDate(ctx, branchName)
}

// GetCommitAuthor returns the commit author for a branch
func (e *engineImpl) GetCommitAuthor(ctx context.Context, branchName string) (string, error) {
	return git.GetCommitAuthor(ctx, branchName)
}

// GetRevision returns the SHA of a branch
func (e *engineImpl) GetRevision(ctx context.Context, branchName string) (string, error) {
	return git.GetRevision(ctx, branchName)
}

// GetPrInfo returns PR information for a branch
func (e *engineImpl) GetPrInfo(ctx context.Context, branchName string) (*PrInfo, error) {
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	if meta.PrInfo == nil {
		return nil, nil
	}

	prInfo := &PrInfo{
		Number:  meta.PrInfo.Number,
		Title:   getStringValue(meta.PrInfo.Title),
		Body:    getStringValue(meta.PrInfo.Body),
		IsDraft: getBoolValue(meta.PrInfo.IsDraft),
		State:   getStringValue(meta.PrInfo.State),
		Base:    getStringValue(meta.PrInfo.Base),
		URL:     getStringValue(meta.PrInfo.URL),
	}

	return prInfo, nil
}

// UpsertPrInfo updates or creates PR information for a branch
func (e *engineImpl) UpsertPrInfo(ctx context.Context, branchName string, prInfo *PrInfo) error {
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		meta = &git.Meta{}
	}

	if meta.PrInfo == nil {
		meta.PrInfo = &git.PrInfo{}
	}

	// Update PR info fields
	if prInfo.Number != nil {
		meta.PrInfo.Number = prInfo.Number
	}
	if prInfo.Title != "" {
		meta.PrInfo.Title = &prInfo.Title
	}
	if prInfo.Body != "" {
		meta.PrInfo.Body = &prInfo.Body
	}
	meta.PrInfo.IsDraft = &prInfo.IsDraft
	if prInfo.State != "" {
		meta.PrInfo.State = &prInfo.State
	}
	if prInfo.Base != "" {
		meta.PrInfo.Base = &prInfo.Base
	}
	if prInfo.URL != "" {
		meta.PrInfo.URL = &prInfo.URL
	}

	return git.WriteMetadataRef(branchName, meta)
}

// GetParentPrecondition returns the parent branch, or trunk if no parent
// This is used for validation where we expect a parent to exist
func (e *engineImpl) GetParentPrecondition(branchName string) string {
	parent := e.GetParent(branchName)
	if parent == "" {
		return e.Trunk()
	}
	return parent
}

// BranchMatchesRemote checks if a branch matches its remote
// For now, always return true (simplified)
func (e *engineImpl) BranchMatchesRemote(ctx context.Context, branchName string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get local branch SHA
	localSha, err := e.GetRevision(ctx, branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get local revision for %s: %w", branchName, err)
	}

	// Get remote SHA from cache
	remoteSha, exists := e.remoteShas[branchName]
	if !exists {
		// No remote branch exists - branch doesn't match remote
		return false, nil
	}

	return localSha == remoteSha, nil
}

// PopulateRemoteShas populates remote branch information by fetching SHAs from remote
func (e *engineImpl) PopulateRemoteShas(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear existing remote SHAs
	e.remoteShas = make(map[string]string)

	// Fetch remote SHAs using git ls-remote
	remote := "origin" // TODO: Get from config
	remoteShas, err := git.FetchRemoteShas(ctx, remote)
	if err != nil {
		// Don't fail if we can't fetch remote SHAs (e.g., offline)
		return nil
	}

	e.remoteShas = remoteShas
	return nil
}

// PushBranch pushes a branch to the remote
func (e *engineImpl) PushBranch(ctx context.Context, branchName string, remote string, force bool, forceWithLease bool) error {
	return git.PushBranch(ctx, branchName, remote, force, forceWithLease)
}

// Reset clears all branch metadata and rebuilds with new trunk
func (e *engineImpl) Reset(ctx context.Context, newTrunkName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update trunk
	e.trunk = newTrunkName

	// Delete all metadata refs
	metadataRefs, err := git.GetMetadataRefList()
	if err != nil {
		return fmt.Errorf("failed to get metadata refs: %w", err)
	}

	for branchName := range metadataRefs {
		if err := git.DeleteMetadataRef(branchName); err != nil {
			// Ignore errors for non-existent refs
			continue
		}
	}

	// Rebuild cache (already holding lock, so call rebuildInternal)
	// Refresh currentBranch since we're resetting everything
	return e.rebuildInternal(true)
}

// Rebuild reloads branch cache with new trunk
func (e *engineImpl) Rebuild(ctx context.Context, newTrunkName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update trunk
	e.trunk = newTrunkName

	// Rebuild cache (already holding lock, so call rebuildInternal)
	// Refresh currentBranch since we might have switched branches
	return e.rebuildInternal(true)
}

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

	// Load metadata for each branch
	for _, branchName := range branches {
		meta, err := git.ReadMetadataRef(branchName)
		if err != nil {
			continue
		}

		if meta.ParentBranchName != nil {
			parent := *meta.ParentBranchName
			e.parentMap[branchName] = parent
			e.childrenMap[parent] = append(e.childrenMap[parent], branchName)
		}
	}

	return nil
}

// TrackBranch tracks a branch with a parent branch
func (e *engineImpl) TrackBranch(ctx context.Context, branchName string, parentBranchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

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

	// Get merge base (parent revision)
	parentRevision, err := git.GetMergeBase(ctx, branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Create metadata
	meta := &git.Meta{
		ParentBranchName:     &parentBranchName,
		ParentBranchRevision: &parentRevision,
	}

	// Write metadata ref
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	// Update in-memory cache
	e.parentMap[branchName] = parentBranchName
	e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)

	return nil
}

// UntrackBranch stops tracking a branch by deleting its metadata
func (e *engineImpl) UntrackBranch(ctx context.Context, branchName string) error {
	if e.IsTrunk(branchName) {
		return fmt.Errorf("cannot untrack trunk branch")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Delete metadata
	if err := git.DeleteMetadataRef(branchName); err != nil {
		return fmt.Errorf("failed to delete metadata ref: %w", err)
	}

	// Rebuild cache (already holding lock, so call rebuildInternal)
	// Refresh currentBranch since we're resetting everything
	return e.rebuildInternal(true)
}

// PullTrunk pulls the trunk branch from remote
func (e *engineImpl) PullTrunk(ctx context.Context) (PullResult, error) {
	remote := git.GetRemote(ctx)
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
	remote := git.GetRemote(ctx)

	e.mu.RLock()
	trunk := e.trunk
	currentBranch := e.currentBranch
	e.mu.RUnlock()

	// Get remote SHA
	remoteSha, err := git.GetRemoteSha(ctx, remote, trunk)
	if err != nil {
		return fmt.Errorf("failed to get remote SHA: %w", err)
	}

	// Checkout trunk
	if err := git.CheckoutBranch(ctx, trunk); err != nil {
		return fmt.Errorf("failed to checkout trunk: %w", err)
	}

	// Hard reset to remote
	if err := git.HardReset(ctx, remoteSha); err != nil {
		// Try to switch back
		if currentBranch != "" {
			_ = git.CheckoutBranch(ctx, currentBranch)
		}
		return fmt.Errorf("failed to reset trunk: %w", err)
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != trunk {
		if err := git.CheckoutBranch(ctx, currentBranch); err != nil {
			return fmt.Errorf("failed to switch back: %w", err)
		}
	}

	// Rebuild to refresh branch cache
	if err := e.rebuild(); err != nil {
		return fmt.Errorf("failed to rebuild after reset: %w", err)
	}

	return nil
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

// RestackBranch rebases a branch onto its parent
// If the parent has been merged/deleted, it will automatically reparent to the nearest valid ancestor
func (e *engineImpl) RestackBranch(ctx context.Context, branchName string) (RestackBranchResult, error) {
	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		return RestackBranchResult{Result: RestackUnneeded}, fmt.Errorf("branch %s is not tracked", branchName)
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
	parentRev, err := e.GetRevision(ctx, parent)
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, fmt.Errorf("failed to get parent revision: %w", err)
	}

	// Check if branch needs restacking
	if e.IsBranchFixed(ctx, branchName) {
		return RestackBranchResult{
			Result:            RestackUnneeded,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, nil
	}

	// Get old parent revision from metadata
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, fmt.Errorf("failed to read metadata: %w", err)
	}

	oldParentRev := parentRev
	if meta.ParentBranchRevision != nil {
		oldParentRev = *meta.ParentBranchRevision
	}

	// If parent hasn't changed, no need to restack
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

	// Rebuild to refresh cache
	if err := e.rebuild(); err != nil {
		return RestackBranchResult{
			Result:            RestackDone,
			RebasedBranchBase: parentRev,
			Reparented:        reparented,
			OldParent:         oldParent,
			NewParent:         parent,
		}, fmt.Errorf("failed to rebuild after restack: %w", err)
	}

	return RestackBranchResult{
		Result:            RestackDone,
		RebasedBranchBase: parentRev,
		Reparented:        reparented,
		OldParent:         oldParent,
		NewParent:         parent,
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

// IsMergedIntoTrunk checks if a branch is merged into trunk
func (e *engineImpl) IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error) {
	e.mu.RLock()
	trunk := e.trunk
	e.mu.RUnlock()
	return git.IsMerged(ctx, branchName, trunk)
}

// IsBranchEmpty checks if a branch has no changes compared to its parent
func (e *engineImpl) IsBranchEmpty(ctx context.Context, branchName string) (bool, error) {
	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	trunk := e.trunk
	e.mu.RUnlock()

	if !ok {
		// If not tracked, compare to trunk
		parent = trunk
	}

	// Get parent revision
	parentRev, err := e.GetRevision(ctx, parent)
	if err != nil {
		return false, fmt.Errorf("failed to get parent revision: %w", err)
	}

	return git.IsDiffEmpty(ctx, branchName, parentRev)
}

// FindMostRecentTrackedAncestors finds the most recent tracked ancestors of a branch
// by checking the branch's commit history against tracked branch tips.
// Returns a slice of branch names that point to the most recent tracked commit in history.
func (e *engineImpl) FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	trunk := e.trunk

	// Map of commit SHA to slice of tracked branch names
	trackedBranchTips := make(map[string][]string)

	// Add trunk tip
	trunkRev, err := git.GetRevision(ctx, trunk)
	if err == nil {
		trackedBranchTips[trunkRev] = append(trackedBranchTips[trunkRev], trunk)
	}

	// Get all tracked branches and their tips
	for _, candidate := range e.branches {
		// Skip the branch itself and trunk (already handled)
		if candidate == branchName || candidate == trunk {
			continue
		}

		// Only consider tracked branches
		if _, ok := e.parentMap[candidate]; !ok {
			continue
		}

		// Skip branches merged into trunk
		if merged, err := git.IsMerged(ctx, candidate, trunk); err == nil && merged {
			continue
		}

		// Get candidate revision
		candidateRev, err := git.GetRevision(ctx, candidate)
		if err != nil {
			continue
		}

		trackedBranchTips[candidateRev] = append(trackedBranchTips[candidateRev], candidate)
	}

	// Get history of the branch we're tracking
	history, err := git.GetCommitHistorySHAs(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch history: %w", err)
	}

	// Iterate through history (newest to oldest) and find the first tracked tip(s)
	for _, sha := range history {
		if ancestors, ok := trackedBranchTips[sha]; ok {
			// Found the most recent tracked commit(s)
			return ancestors, nil
		}
	}

	return nil, fmt.Errorf("no tracked ancestor found for branch %s", branchName)
}

// DeleteBranch deletes a branch and its metadata
func (e *engineImpl) DeleteBranch(ctx context.Context, branchName string) error {
	if e.IsTrunk(branchName) {
		return fmt.Errorf("cannot delete trunk branch")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Get children before deletion
	children := e.childrenMap[branchName]

	// Get parent
	parent, ok := e.parentMap[branchName]
	if !ok {
		parent = e.trunk
	}

	// If deleting current branch, switch to trunk first
	if branchName == e.currentBranch {
		if err := git.CheckoutBranch(ctx, e.trunk); err != nil {
			return fmt.Errorf("failed to switch to trunk before deleting current branch: %w", err)
		}
		e.currentBranch = e.trunk
	}

	// Delete git branch
	if err := git.DeleteBranch(ctx, branchName); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// Delete metadata
	_ = git.DeleteMetadataRef(branchName)

	// Update children to point to parent
	for _, child := range children {
		if err := e.setParentInternal(ctx, child, parent); err != nil {
			// Log error but continue
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

// SetParent updates a branch's parent
func (e *engineImpl) SetParent(ctx context.Context, branchName string, parentBranchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.setParentInternal(ctx, branchName, parentBranchName)
}

// setParentInternal updates parent without locking (caller must hold lock)
func (e *engineImpl) setParentInternal(ctx context.Context, branchName string, parentBranchName string) error {
	// Get new parent revision
	parentRev, err := git.GetMergeBase(ctx, branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Read existing metadata
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		meta = &git.Meta{}
	}

	// Update parent
	oldParent := ""
	if meta.ParentBranchName != nil {
		oldParent = *meta.ParentBranchName
	}

	meta.ParentBranchName = &parentBranchName
	meta.ParentBranchRevision = &parentRev

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

// GetRelativeStackUpstack returns all branches in the upstack (descendants)
func (e *engineImpl) GetRelativeStackUpstack(branchName string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

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

// SquashCurrentBranch squashes all commits in the current branch into a single commit
func (e *engineImpl) SquashCurrentBranch(ctx context.Context, opts SquashOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get current branch
	branchName := e.currentBranch
	if branchName == "" {
		return fmt.Errorf("not on a branch")
	}

	// Check if branch is trunk (check directly since we hold the lock)
	if branchName == e.trunk {
		return fmt.Errorf("cannot squash trunk branch")
	}

	// Read metadata to get parent branch revision
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if meta.ParentBranchRevision == nil {
		return fmt.Errorf("branch has no parent revision")
	}

	parentBranchRevision := *meta.ParentBranchRevision

	// Get current branch revision
	branchRevision, err := e.GetRevision(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Get commit range SHAs from parent to current branch
	commitSHAs, err := git.GetCommitRangeSHAs(ctx, parentBranchRevision, branchRevision)
	if err != nil {
		return fmt.Errorf("failed to get commit range: %w", err)
	}

	// Check if there are commits to squash
	if len(commitSHAs) == 0 {
		return fmt.Errorf("no commits to squash")
	}

	// Get the last (oldest) commit SHA from the range
	// git log returns commits in reverse chronological order (newest first)
	// So the last element is the oldest commit
	oldestCommitSHA := commitSHAs[len(commitSHAs)-1]

	// Soft reset to the oldest commit (keeps all changes staged)
	// This moves HEAD to the oldest commit, staging all changes from newer commits
	if err := git.SoftReset(ctx, oldestCommitSHA); err != nil {
		return fmt.Errorf("failed to soft reset: %w", err)
	}

	// Commit with amend flag to modify the oldest commit to include all changes
	// This is correct: we reset to the oldest commit, then amend it to include all subsequent changes
	// Only pass noEdit and message, let git handle editor by default
	commitOpts := git.CommitOptions{
		Amend:   true,
		Message: opts.Message,
		NoEdit:  opts.NoEdit,
		// Don't set Edit - git will open editor by default if no message and no noEdit
	}

	if err := git.CommitWithOptions(commitOpts); err != nil {
		// Try to rollback on error
		if rollbackErr := git.SoftReset(ctx, branchRevision); rollbackErr != nil {
			// Log rollback error but return original error
			return fmt.Errorf("failed to commit and failed to rollback: commit error: %w, rollback error: %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Rebuild to refresh cache (parent/children relationships remain the same)
	// Don't refresh currentBranch - we're still on the same branch
	if err := e.rebuildInternal(false); err != nil {
		return fmt.Errorf("failed to rebuild after squash: %w", err)
	}

	return nil
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

// FindBranchForCommit finds which branch a commit belongs to
func (e *engineImpl) FindBranchForCommit(ctx context.Context, commitSHA string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, branchName := range e.branches {
		commits, err := e.GetAllCommits(ctx, branchName, CommitFormatSHA)
		if err != nil {
			continue
		}

		for _, sha := range commits {
			if sha == commitSHA {
				return branchName, nil
			}
		}
	}

	return "", fmt.Errorf("commit %s not found in any branch", commitSHA)
}

// GetAllCommits returns commits for a branch in various formats
func (e *engineImpl) GetAllCommits(ctx context.Context, branchName string, format CommitFormat) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if branch is trunk
	if branchName == e.trunk {
		// For trunk, get just the one commit
		branchRevision, err := e.GetRevision(ctx, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to get branch revision: %w", err)
		}
		return git.GetCommitRange("", branchRevision, string(format))
	}

	// Get metadata to find parent revision
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Get branch revision
	branchRevision, err := e.GetRevision(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Get parent revision (base)
	var baseRevision string
	if meta.ParentBranchRevision != nil {
		baseRevision = *meta.ParentBranchRevision
	}

	return git.GetCommitRange(baseRevision, branchRevision, string(format))
}

// ApplySplitToCommits creates branches at specified commit points
func (e *engineImpl) ApplySplitToCommits(ctx context.Context, opts ApplySplitOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(opts.BranchNames) != len(opts.BranchPoints) {
		return fmt.Errorf("invalid number of branch names: got %d names but %d branch points", len(opts.BranchNames), len(opts.BranchPoints))
	}

	// Get metadata for the branch being split
	meta, err := git.ReadMetadataRef(opts.BranchToSplit)
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
		branchRevision, err := git.GetCommitSHA(opts.BranchToSplit, reversedBranchPoints[idx])
		if err != nil {
			return fmt.Errorf("failed to get commit SHA at offset %d: %w", reversedBranchPoints[idx], err)
		}

		// Create branch at that SHA
		_, err = git.RunGitCommandWithContext(ctx, "branch", "-f", branchName, branchRevision)
		if err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}

		// Preserve PR info if branch name matches original
		var prInfo *PrInfo
		if branchName == opts.BranchToSplit {
			prInfo, _ = e.GetPrInfo(ctx, opts.BranchToSplit)
		}

		// Track branch with parent
		newMeta := &git.Meta{
			ParentBranchName:     &lastBranchName,
			ParentBranchRevision: &lastBranchRevision,
		}

		// Preserve PR info if applicable
		if prInfo != nil {
			newMeta.PrInfo = &git.PrInfo{
				Number:  prInfo.Number,
				Title:   stringPtr(prInfo.Title),
				Body:    stringPtr(prInfo.Body),
				IsDraft: boolPtr(prInfo.IsDraft),
				State:   stringPtr(prInfo.State),
				Base:    stringPtr(prInfo.Base),
				URL:     stringPtr(prInfo.URL),
			}
		}

		if err := git.WriteMetadataRef(branchName, newMeta); err != nil {
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
		for _, childBranchName := range children {
			if err := e.SetParent(ctx, childBranchName, lastBranchName); err != nil {
				return fmt.Errorf("failed to update parent for %s: %w", childBranchName, err)
			}
		}
	}

	// Delete original branch if not in branchNames
	if !contains(opts.BranchNames, opts.BranchToSplit) {
		if err := e.DeleteBranch(ctx, opts.BranchToSplit); err != nil {
			return fmt.Errorf("failed to delete original branch: %w", err)
		}
	}

	// Checkout last branch
	e.currentBranch = lastBranchName
	if err := git.CheckoutBranch(ctx, lastBranchName); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", lastBranchName, err)
	}

	return nil
}

// Detach detaches HEAD to a specific revision
func (e *engineImpl) Detach(ctx context.Context, revision string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Checkout the revision in detached HEAD state
	_, err := git.RunGitCommandWithContext(ctx, "checkout", "--detach", revision)
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
	branchRevision, err := e.GetRevision(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Get the parent branch
	parentBranchName := e.parentMap[branchName]
	if parentBranchName == "" {
		parentBranchName = e.trunk
	}

	// Get the merge base between this branch and its parent
	mergeBase, err := git.GetMergeBase(ctx, branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Detach HEAD to the branch revision first
	_, err = git.RunGitCommandWithContext(ctx, "checkout", "--detach", branchRevision)
	if err != nil {
		return fmt.Errorf("failed to detach HEAD: %w", err)
	}

	// Soft reset to the merge base - this keeps all the branch's changes
	// but unstages them, allowing the user to re-stage them interactively
	_, err = git.RunGitCommandWithContext(ctx, "reset", mergeBase)
	if err != nil {
		return fmt.Errorf("failed to soft reset: %w", err)
	}

	e.currentBranch = ""
	return nil
}

// CheckoutBranch checks out a branch
func (e *engineImpl) ForceCheckoutBranch(ctx context.Context, branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := git.RunGitCommandWithContext(ctx, "checkout", "-f", branchName)
	if err != nil {
		return fmt.Errorf("failed to force checkout branch: %w", err)
	}

	e.currentBranch = branchName
	return nil
}

// Helper functions
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
