package engine

import (
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
	if err := e.rebuild(); err != nil {
		return nil, fmt.Errorf("failed to rebuild engine: %w", err)
	}

	return e, nil
}

// rebuild loads all branches and their metadata from Git
func (e *engineImpl) rebuild() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rebuildInternal()
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

	// Add ancestors if RecursiveParents is true
	if scope.RecursiveParents {
		current := branchName
		ancestors := []string{}
		for {
			if current == e.trunk {
				break
			}
			parent, ok := e.parentMap[current]
			if !ok {
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
func (e *engineImpl) IsBranchFixed(branchName string) bool {
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
	parentRev, err := e.GetRevision(parent)
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
func (e *engineImpl) GetCommitDate(branchName string) (time.Time, error) {
	return git.GetCommitDate(branchName)
}

// GetCommitAuthor returns the commit author for a branch
func (e *engineImpl) GetCommitAuthor(branchName string) (string, error) {
	return git.GetCommitAuthor(branchName)
}

// GetRevision returns the SHA of a branch
func (e *engineImpl) GetRevision(branchName string) (string, error) {
	return git.GetRevision(branchName)
}

// GetPrInfo returns PR information for a branch
func (e *engineImpl) GetPrInfo(branchName string) (*PrInfo, error) {
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
func (e *engineImpl) UpsertPrInfo(branchName string, prInfo *PrInfo) error {
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
func (e *engineImpl) BranchMatchesRemote(branchName string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get local branch SHA
	localSha, err := git.GetRevision(branchName)
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
func (e *engineImpl) PopulateRemoteShas() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear existing remote SHAs
	e.remoteShas = make(map[string]string)

	// Fetch remote SHAs using git ls-remote
	remote := "origin" // TODO: Get from config
	remoteShas, err := git.FetchRemoteShas(remote)
	if err != nil {
		// Don't fail if we can't fetch remote SHAs (e.g., offline)
		return nil
	}

	e.remoteShas = remoteShas
	return nil
}

// Reset clears all branch metadata and rebuilds with new trunk
func (e *engineImpl) Reset(newTrunkName string) error {
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
	return e.rebuildInternal()
}

// Rebuild reloads branch cache with new trunk
func (e *engineImpl) Rebuild(newTrunkName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update trunk
	e.trunk = newTrunkName

	// Rebuild cache (already holding lock, so call rebuildInternal)
	return e.rebuildInternal()
}

// rebuildInternal is the internal rebuild logic without locking
func (e *engineImpl) rebuildInternal() error {
	// Get all branch names
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}
	e.branches = branches

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
func (e *engineImpl) TrackBranch(branchName string, parentBranchName string) error {
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
			return fmt.Errorf("parent branch %s does not exist", parentBranchName)
		}
	}

	// Get merge base (parent revision)
	parentRevision, err := git.GetMergeBase(branchName, parentBranchName)
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

// PullTrunk pulls the trunk branch from remote
func (e *engineImpl) PullTrunk() (PullResult, error) {
	remote := git.GetRemote()
	e.mu.RLock()
	trunk := e.trunk
	e.mu.RUnlock()
	gitResult, err := git.PullBranch(remote, trunk)
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
func (e *engineImpl) ResetTrunkToRemote() error {
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
	if err := git.CheckoutBranch(trunk); err != nil {
		return fmt.Errorf("failed to checkout trunk: %w", err)
	}

	// Hard reset to remote
	if err := git.HardReset(remoteSha); err != nil {
		// Try to switch back
		if currentBranch != "" {
			_ = git.CheckoutBranch(currentBranch)
		}
		return fmt.Errorf("failed to reset trunk: %w", err)
	}

	// Switch back to original branch
	if currentBranch != "" && currentBranch != trunk {
		if err := git.CheckoutBranch(currentBranch); err != nil {
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
func (e *engineImpl) RestackBranch(branchName string) (RestackBranchResult, error) {
	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		return RestackBranchResult{Result: RestackUnneeded}, fmt.Errorf("branch %s is not tracked", branchName)
	}

	// Get parent revision (needed for rebasedBranchBase even if restack is unneeded)
	parentRev, err := e.GetRevision(parent)
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, fmt.Errorf("failed to get parent revision: %w", err)
	}

	// Check if branch needs restacking
	if e.IsBranchFixed(branchName) {
		return RestackBranchResult{Result: RestackUnneeded, RebasedBranchBase: parentRev}, nil
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
		return RestackBranchResult{Result: RestackUnneeded, RebasedBranchBase: parentRev}, nil
	}

	// Perform rebase
	gitResult, err := git.Rebase(branchName, parent, oldParentRev)
	if err != nil {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, err
	}

	if gitResult == git.RebaseConflict {
		return RestackBranchResult{Result: RestackConflict, RebasedBranchBase: parentRev}, nil
	}

	// Update metadata with new parent revision
	meta.ParentBranchRevision = &parentRev
	if err := git.WriteMetadataRef(branchName, meta); err != nil {
		return RestackBranchResult{Result: RestackDone, RebasedBranchBase: parentRev}, fmt.Errorf("failed to update metadata: %w", err)
	}

	// Rebuild to refresh cache
	if err := e.rebuild(); err != nil {
		return RestackBranchResult{Result: RestackDone, RebasedBranchBase: parentRev}, fmt.Errorf("failed to rebuild after restack: %w", err)
	}

	return RestackBranchResult{Result: RestackDone, RebasedBranchBase: parentRev}, nil
}

// ContinueRebase continues an in-progress rebase
func (e *engineImpl) ContinueRebase(rebasedBranchBase string) (ContinueRebaseResult, error) {
	// Call git rebase --continue
	result, err := git.RebaseContinue()
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
func (e *engineImpl) IsMergedIntoTrunk(branchName string) (bool, error) {
	e.mu.RLock()
	trunk := e.trunk
	e.mu.RUnlock()
	return git.IsMerged(branchName, trunk)
}

// IsBranchEmpty checks if a branch has no changes compared to its parent
func (e *engineImpl) IsBranchEmpty(branchName string) (bool, error) {
	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	trunk := e.trunk
	e.mu.RUnlock()

	if !ok {
		// If not tracked, compare to trunk
		parent = trunk
	}

	// Get parent revision
	parentRev, err := e.GetRevision(parent)
	if err != nil {
		return false, fmt.Errorf("failed to get parent revision: %w", err)
	}

	return git.IsDiffEmpty(branchName, parentRev)
}

// DeleteBranch deletes a branch and its metadata
func (e *engineImpl) DeleteBranch(branchName string) error {
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

	// Delete git branch
	if err := git.DeleteBranch(branchName); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// Delete metadata
	if err := git.DeleteMetadataRef(branchName); err != nil {
		// Non-fatal, continue
	}

	// Update children to point to parent
	for _, child := range children {
		if err := e.setParentInternal(child, parent); err != nil {
			// Log error but continue
			continue
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

	// Update children map for parent
	if parent != "" {
		parentChildren := e.childrenMap[parent]
		for i, c := range parentChildren {
			if c == branchName {
				e.childrenMap[parent] = append(parentChildren[:i], parentChildren[i+1:]...)
				break
			}
		}
		// Add children to parent
		e.childrenMap[parent] = append(e.childrenMap[parent], children...)
	}

	return nil
}

// SetParent updates a branch's parent
func (e *engineImpl) SetParent(branchName string, parentBranchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.setParentInternal(branchName, parentBranchName)
}

// setParentInternal updates parent without locking (caller must hold lock)
func (e *engineImpl) setParentInternal(branchName string, parentBranchName string) error {
	// Get new parent revision
	parentRev, err := git.GetMergeBase(branchName, parentBranchName)
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
func (e *engineImpl) SquashCurrentBranch(opts SquashOptions) error {
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
	branchRevision, err := git.GetRevision(branchName)
	if err != nil {
		return fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Get commit range SHAs from parent to current branch
	commitSHAs, err := git.GetCommitRangeSHAs(parentBranchRevision, branchRevision)
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
	if err := git.SoftReset(oldestCommitSHA); err != nil {
		return fmt.Errorf("failed to soft reset: %w", err)
	}

	// Commit with amend flag to modify the oldest commit to include all changes
	// This is correct: we reset to the oldest commit, then amend it to include all subsequent changes
	// Match charcoal behavior: only pass noEdit and message, let git handle editor by default
	commitOpts := git.CommitOptions{
		Amend:   true,
		Message: opts.Message,
		NoEdit:  opts.NoEdit,
		// Don't set Edit - git will open editor by default if no message and no noEdit
	}

	if err := git.CommitWithOptions(commitOpts); err != nil {
		// Try to rollback on error
		if rollbackErr := git.SoftReset(branchRevision); rollbackErr != nil {
			// Log rollback error but return original error
			return fmt.Errorf("failed to commit and failed to rollback: commit error: %w, rollback error: %v", err, rollbackErr)
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Rebuild to refresh cache (parent/children relationships remain the same)
	if err := e.rebuildInternal(); err != nil {
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

// GetAllCommits returns commits for a branch in various formats
func (e *engineImpl) GetAllCommits(branchName string, format CommitFormat) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if branch is trunk
	if branchName == e.trunk {
		// For trunk, get just the one commit
		branchRevision, err := git.GetRevision(branchName)
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
	branchRevision, err := git.GetRevision(branchName)
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
func (e *engineImpl) ApplySplitToCommits(opts ApplySplitOptions) error {
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
		_, err = git.RunGitCommand("branch", "-f", branchName, branchRevision)
		if err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}

		// Preserve PR info if branch name matches original
		var prInfo *PrInfo
		if branchName == opts.BranchToSplit {
			prInfo, _ = e.GetPrInfo(opts.BranchToSplit)
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
			if err := e.SetParent(childBranchName, lastBranchName); err != nil {
				return fmt.Errorf("failed to update parent for %s: %w", childBranchName, err)
			}
		}
	}

	// Delete original branch if not in branchNames
	if !contains(opts.BranchNames, opts.BranchToSplit) {
		if err := e.DeleteBranch(opts.BranchToSplit); err != nil {
			return fmt.Errorf("failed to delete original branch: %w", err)
		}
	}

	// Checkout last branch
	e.currentBranch = lastBranchName
	if err := git.CheckoutBranch(lastBranchName); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", lastBranchName, err)
	}

	return nil
}

// Detach detaches HEAD to a specific revision
func (e *engineImpl) Detach(revision string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Checkout the revision in detached HEAD state
	_, err := git.RunGitCommand("checkout", "--detach", revision)
	if err != nil {
		return fmt.Errorf("failed to detach HEAD: %w", err)
	}

	e.currentBranch = ""
	return nil
}

// DetachAndResetBranchChanges detaches and resets branch changes
func (e *engineImpl) DetachAndResetBranchChanges(branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get branch revision
	branchRevision, err := git.GetRevision(branchName)
	if err != nil {
		return fmt.Errorf("failed to get branch revision: %w", err)
	}

	// Detach and reset
	_, err = git.RunGitCommand("checkout", "--detach", branchRevision)
	if err != nil {
		return fmt.Errorf("failed to detach HEAD: %w", err)
	}

	// Reset to discard any changes
	if err := git.HardReset(branchRevision); err != nil {
		return fmt.Errorf("failed to reset: %w", err)
	}

	e.currentBranch = ""
	return nil
}

// ForceCheckoutBranch force checks out a branch
func (e *engineImpl) ForceCheckoutBranch(branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := git.RunGitCommand("checkout", "-f", branchName)
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
