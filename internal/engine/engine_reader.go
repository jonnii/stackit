package engine

import (
	"context"
	"fmt"
	"iter"
	"time"

	"stackit.dev/stackit/internal/git"
)

// AllBranchNames returns all branch names
func (e *engineImpl) AllBranchNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.branches
}

// CurrentBranch returns the current branch name
func (e *engineImpl) CurrentBranch() string {
	e.mu.Lock()
	if current, err := git.GetCurrentBranch(); err == nil {
		e.currentBranch = current
	} else {
		// Not on a branch (e.g., detached HEAD)
		e.currentBranch = ""
	}
	e.mu.Unlock()

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

// IsBranchUpToDate checks if a branch is up to date with its parent
// A branch is up to date if its parent revision matches the stored parent revision
func (e *engineImpl) IsBranchUpToDate(branchName string) bool {
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
	parentRev, err := git.GetRevision(parent)
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
func (e *engineImpl) BranchMatchesRemote(branchName string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get local branch SHA
	localSha, err := e.GetRevision(branchName)
	if err != nil {
		return false, nil
	}

	// Get remote SHA from cache
	remoteSha, exists := e.remoteShas[branchName]
	if !exists {
		// No remote branch exists - branch doesn't match remote
		return false, nil
	}

	return localSha == remoteSha, nil
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
	parentRev, err := e.GetRevision(parent)
	if err != nil {
		return false, err
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
	trunkRev, err := git.GetRevision(trunk)
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
		candidateRev, err := git.GetRevision(candidate)
		if err != nil {
			continue
		}

		trackedBranchTips[candidateRev] = append(trackedBranchTips[candidateRev], candidate)
	}

	// Get history of the branch we're tracking
	history, err := git.GetCommitHistorySHAs(branchName)
	if err != nil {
		return nil, err
	}

	// Iterate through history (newest to oldest) and find the first tracked tip(s)
	for _, sha := range history {
		if ancestors, ok := trackedBranchTips[sha]; ok {
			// Found the most recent tracked commit(s)
			return ancestors, nil
		}
	}

	return nil, nil
}

// FindBranchForCommit finds which branch a commit belongs to
func (e *engineImpl) FindBranchForCommit(commitSHA string) (string, error) {
	e.mu.RLock()
	branches := make([]string, len(e.branches))
	copy(branches, e.branches)
	e.mu.RUnlock()

	for _, branchName := range branches {
		commits, err := e.GetAllCommits(branchName, CommitFormatSHA)
		if err != nil {
			continue
		}

		for _, sha := range commits {
			if sha == commitSHA {
				return branchName, nil
			}
		}
	}

	return "", nil
}

// GetAllCommits returns commits for a branch in various formats
func (e *engineImpl) GetAllCommits(branchName string, format CommitFormat) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if branch is trunk
	if branchName == e.trunk {
		// For trunk, get just the one commit
		branchRevision, err := e.GetRevision(branchName)
		if err != nil {
			return nil, err
		}
		return git.GetCommitRange("", branchRevision, string(format))
	}

	// Get metadata to find parent revision
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	// Get branch revision
	branchRevision, err := e.GetRevision(branchName)
	if err != nil {
		return nil, err
	}

	// Get parent revision (base)
	var baseRevision string
	if meta.ParentBranchRevision != nil {
		baseRevision = *meta.ParentBranchRevision
	}

	return git.GetCommitRange(baseRevision, branchRevision, string(format))
}

// GetRelativeStackUpstack returns all branches in the upstack (descendants)
func (e *engineImpl) GetRelativeStackUpstack(branchName string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.getRelativeStackUpstackInternal(branchName)
}

// GetRelativeStackDownstack returns all branches in the downstack (ancestors)
func (e *engineImpl) GetRelativeStackDownstack(branchName string) []string {
	scope := Scope{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	return e.GetRelativeStack(branchName, scope)
}

// GetFullStack returns the entire stack containing the branch
func (e *engineImpl) GetFullStack(branchName string) []string {
	scope := Scope{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: true,
	}
	return e.GetRelativeStack(branchName, scope)
}

// SortBranchesTopologically sorts branches so parents come before children.
// This ensures correct restack order (bottom of stack first).
func (e *engineImpl) SortBranchesTopologically(branches []string) []string {
	if len(branches) == 0 {
		return branches
	}

	// Calculate depth for each branch (distance from trunk)
	depths := make(map[string]int)
	var getDepth func(branch string) int
	getDepth = func(branch string) int {
		if depth, ok := depths[branch]; ok {
			return depth
		}

		e.mu.RLock()
		isTrunk := branch == e.trunk
		parent := e.parentMap[branch]
		e.mu.RUnlock()

		if isTrunk {
			depths[branch] = 0
			return 0
		}
		if parent == "" || e.IsTrunk(parent) {
			depths[branch] = 1
			return 1
		}
		depths[branch] = getDepth(parent) + 1
		return depths[branch]
	}

	// Calculate depth for all branches
	for _, branch := range branches {
		getDepth(branch)
	}

	// Sort by depth (parents first, then children)
	result := make([]string, len(branches))
	copy(result, branches)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if depths[result[i]] > depths[result[j]] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetDeletionStatus checks if a branch should be deleted
func (e *engineImpl) GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error) {
	// Check PR info
	prInfo, err := e.GetPrInfo(branchName)
	if err == nil && prInfo != nil {
		const (
			prStateClosed = "CLOSED"
			prStateMerged = "MERGED"
		)
		if prInfo.State == prStateClosed {
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is closed on GitHub", branchName)}, nil
		}
		if prInfo.State == prStateMerged {
			base := prInfo.Base
			if base == "" {
				base = e.Trunk()
			}
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is merged into %s", branchName, base)}, nil
		}
	}

	// Check if merged into trunk
	merged, err := e.IsMergedIntoTrunk(ctx, branchName)
	if err == nil && merged {
		return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is merged into %s", branchName, e.Trunk())}, nil
	}

	// Check if empty
	empty, err := e.IsBranchEmpty(ctx, branchName)
	if err == nil && empty {
		// Only delete empty branches if they have a PR
		if prInfo != nil && prInfo.Number != nil {
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is empty", branchName)}, nil
		}
	}

	return DeletionStatus{SafeToDelete: false, Reason: ""}, nil
}

// BranchesDepthFirst returns an iterator that yields branches starting from startBranch in depth-first order.
// Each iteration yields (branchName, depth) where depth is 0 for the start branch.
// The iterator can be used with range loops and supports early termination with break.
func (e *engineImpl) BranchesDepthFirst(startBranch string) iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		visited := make(map[string]bool)
		var visit func(branch string, depth int) bool
		visit = func(branch string, depth int) bool {
			if visited[branch] {
				return true // cycle detection
			}
			visited[branch] = true

			if !yield(branch, depth) {
				return false // iterator wants to stop
			}

			children := e.GetChildren(branch)
			for _, child := range children {
				if !visit(child, depth+1) {
					return false
				}
			}
			return true
		}

		visit(startBranch, 0)
	}
}
