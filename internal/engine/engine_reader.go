package engine

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"stackit.dev/stackit/internal/git"
)

// AllBranches returns all branches
func (e *engineImpl) AllBranches() []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	branches := make([]Branch, len(e.branches))
	for i, name := range e.branches {
		branches[i] = NewBranch(name, e)
	}
	return branches
}

// CurrentBranch returns the current branch (nil if not on a branch)
func (e *engineImpl) CurrentBranch() *Branch {
	e.mu.Lock()
	if current, err := e.git.GetCurrentBranch(); err == nil {
		e.currentBranch = current
	} else {
		// Not on a branch (e.g., detached HEAD)
		e.currentBranch = ""
	}
	e.mu.Unlock()

	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentBranch == "" {
		return nil
	}
	branch := NewBranch(e.currentBranch, e)
	return &branch
}

// Trunk returns the trunk branch
func (e *engineImpl) Trunk() Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return NewBranch(e.trunk, e)
}

// GetBranch returns a Branch wrapper for the given branch name
func (e *engineImpl) GetBranch(branchName string) Branch {
	return NewBranch(branchName, e)
}

// GetParent returns the parent branch (nil if no parent)
func (e *engineImpl) GetParent(branch Branch) *Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if parent, ok := e.parentMap[branch.GetName()]; ok {
		b := NewBranch(parent, e)
		return &b
	}
	return nil
}

// GetChildrenInternal returns the children branches (internal method for Branch type)
func (e *engineImpl) GetChildrenInternal(branchName string) []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if children, ok := e.childrenMap[branchName]; ok {
		branches := make([]Branch, len(children))
		for i, name := range children {
			branches[i] = NewBranch(name, e)
		}
		return branches
	}
	return []Branch{}
}

// GetRelativeStack returns the stack relative to a branch
// Returns branches in order: ancestors (if RecursiveParents), current (if IncludeCurrent), descendants (if RecursiveChildren)
func (e *engineImpl) GetRelativeStack(branch Branch, rng StackRange) []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []Branch{}

	// Add ancestors if RecursiveParents is true (excluding trunk)
	if rng.RecursiveParents {
		current := branch.GetName()
		ancestors := []Branch{}
		for {
			if current == e.trunk {
				break
			}
			parent, ok := e.parentMap[current]
			if !ok || parent == e.trunk {
				break
			}
			ancestors = append([]Branch{NewBranch(parent, e)}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch if IncludeCurrent is true
	if rng.IncludeCurrent {
		result = append(result, branch)
	}

	// Add descendants if RecursiveChildren is true
	if rng.RecursiveChildren {
		descendants := e.getRelativeStackUpstackInternal(branch.GetName())
		result = append(result, descendants...)
	}

	return result
}

// GetRelativeStackInternal returns the stack relative to a branch (internal method used by Branch type)
// Returns branches in order: ancestors (if RecursiveParents), current (if IncludeCurrent), descendants (if RecursiveChildren)
func (e *engineImpl) GetRelativeStackInternal(branchName string, rng StackRange) []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []Branch{}

	// Add ancestors if RecursiveParents is true (excluding trunk)
	if rng.RecursiveParents {
		current := branchName
		ancestors := []Branch{}
		for {
			if current == e.trunk {
				break
			}
			parent, ok := e.parentMap[current]
			if !ok || parent == e.trunk {
				break
			}
			ancestors = append([]Branch{NewBranch(parent, e)}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch if IncludeCurrent is true
	if rng.IncludeCurrent {
		result = append(result, NewBranch(branchName, e))
	}

	// Add descendants if RecursiveChildren is true
	if rng.RecursiveChildren {
		descendants := e.getRelativeStackUpstackInternal(branchName)
		result = append(result, descendants...)
	}

	return result
}

// IsTrunkInternal checks if a branch is the trunk (internal method used by Branch type)
func (e *engineImpl) IsTrunkInternal(branchName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return branchName == e.trunk
}

// IsBranchTrackedInternal checks if a branch is tracked (has metadata) (internal method used by Branch type)
func (e *engineImpl) IsBranchTrackedInternal(branchName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.parentMap[branchName]
	return ok
}

// GetScopeInternal returns the scope for a branch, inheriting from parent if not set (internal method used by Branch type)
func (e *engineImpl) GetScopeInternal(branchName string) Scope {
	e.mu.RLock()
	defer e.mu.RUnlock()

	current := branchName
	for {
		if scopeStr, ok := e.scopeMap[current]; ok && scopeStr != "" {
			scope := NewScope(scopeStr)
			if scope.IsNone() {
				return Empty()
			}
			return scope
		}
		parent, ok := e.parentMap[current]
		if !ok || parent == e.trunk {
			break
		}
		current = parent
	}
	return Empty()
}

// GetExplicitScopeInternal returns the explicit scope set for a branch (no inheritance)
func (e *engineImpl) GetExplicitScopeInternal(branchName string) Scope {
	e.mu.RLock()
	defer e.mu.RUnlock()

	scopeStr := e.scopeMap[branchName]
	if scopeStr == "" {
		return Empty()
	}
	return NewScope(scopeStr)
}

// IsBranchUpToDateInternal checks if a branch is up to date with its parent
// A branch is up to date if its parent revision matches the stored parent revision
func (e *engineImpl) IsBranchUpToDateInternal(branchName string) bool {
	if e.IsTrunkInternal(branchName) {
		return true
	}

	e.mu.RLock()
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		return true // Not tracked, consider it fixed
	}

	// Get current parent revision
	parentRev, err := e.git.GetRevision(parent)
	if err != nil {
		return false // Can't determine, assume needs restack
	}

	// Get stored parent revision from metadata
	meta, err := e.readMetadataRef(branchName)
	if err != nil {
		return false // No metadata, assume needs restack
	}

	if meta.ParentBranchRevision == nil {
		return false // No stored revision, needs restack
	}

	// Branch is fixed if stored revision matches current parent revision
	return *meta.ParentBranchRevision == parentRev
}

// GetCommitDateInternal returns the commit date for a branch
func (e *engineImpl) GetCommitDateInternal(branchName string) (time.Time, error) {
	return e.git.GetCommitDate(branchName)
}

// GetCommitAuthorInternal returns the commit author for a branch
func (e *engineImpl) GetCommitAuthorInternal(branchName string) (string, error) {
	return e.git.GetCommitAuthor(branchName)
}

// GetRevisionInternal returns the SHA of a branch
func (e *engineImpl) GetRevisionInternal(branchName string) (string, error) {
	return e.git.GetRevision(branchName)
}

// GetCommitCountInternal returns the number of commits for a branch
func (e *engineImpl) GetCommitCountInternal(branchName string) (int, error) {
	e.mu.RLock()
	trunk := e.trunk
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		parent = trunk
	}

	// Get base revision (stored parent revision)
	meta, err := e.readMetadataRef(branchName)
	var base string
	if err == nil && meta.ParentBranchRevision != nil {
		base = *meta.ParentBranchRevision
	} else {
		// Fallback to current parent branch tip if metadata is missing
		baseRev, err := e.git.GetRevision(parent)
		if err != nil {
			return 0, err
		}
		base = baseRev
	}

	branchRev, err := e.git.GetRevision(branchName)
	if err != nil {
		return 0, err
	}

	// If revisions are same, count is 0
	if branchRev == base {
		return 0, nil
	}

	// For real git, we'd use a git helper. I'll use git.GetCommitRange count.

	commits, err := e.GetAllCommitsInternal(branchName, CommitFormatSHA)
	if err != nil {
		return 0, err
	}
	return len(commits), nil
}

// GetDiffStatsInternal returns diff stats for a branch
func (e *engineImpl) GetDiffStatsInternal(branchName string) (int, int, error) {
	e.mu.RLock()
	trunk := e.trunk
	parent, ok := e.parentMap[branchName]
	e.mu.RUnlock()

	if !ok {
		parent = trunk
	}

	// Get base revision (stored parent revision)
	meta, err := e.readMetadataRef(branchName)
	var base string
	if err == nil && meta.ParentBranchRevision != nil {
		base = *meta.ParentBranchRevision
	} else {
		baseRev, err := e.git.GetRevision(parent)
		if err != nil {
			return 0, 0, err
		}
		base = baseRev
	}

	branchRev, err := e.git.GetRevision(branchName)
	if err != nil {
		return 0, 0, err
	}

	// If revisions are same, stats are 0
	if branchRev == base {
		return 0, 0, nil
	}

	// Use git diff --numstat
	output, err := e.git.RunGitCommand("diff", "--numstat", base, branchRev)
	if err != nil {
		return 0, 0, err
	}

	added, deleted := 0, 0
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			var a, d int
			_, _ = fmt.Sscanf(fields[0], "%d", &a)
			_, _ = fmt.Sscanf(fields[1], "%d", &d)
			added += a
			deleted += d
		}
	}

	return added, deleted, nil
}

// BranchMatchesRemote checks if a branch matches its remote
func (e *engineImpl) BranchMatchesRemote(branchName string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get local branch SHA
	localSha, err := e.GetRevisionInternal(branchName)
	if err != nil {
		return false, nil
	}

	// First try to get remote SHA from cache (populated by PopulateRemoteShas)
	remoteSha, exists := e.remoteShas[branchName]
	if exists {
		return localSha == remoteSha, nil
	}

	// Fall back to checking local remote tracking branch (like getBranchRemoteDifference does)
	// This handles cases where remote fetching failed but we have local remote tracking
	remoteTrackingSha, err := e.git.GetRemoteRevision(branchName)
	if err != nil {
		// No remote tracking branch exists
		return false, nil
	}

	return localSha == remoteTrackingSha, nil
}

// IsMergedIntoTrunk checks if a branch is merged into trunk
func (e *engineImpl) IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error) {
	e.mu.RLock()
	trunk := e.trunk
	e.mu.RUnlock()
	return e.git.IsMerged(ctx, branchName, trunk)
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
	parentRev, err := e.GetRevisionInternal(parent)
	if err != nil {
		return false, err
	}

	return e.git.IsDiffEmpty(ctx, branchName, parentRev)
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
	trunkRev, err := e.git.GetRevision(trunk)
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
		if merged, err := e.git.IsMerged(ctx, candidate, trunk); err == nil && merged {
			continue
		}

		// Get candidate revision
		candidateRev, err := e.git.GetRevision(candidate)
		if err != nil {
			continue
		}

		trackedBranchTips[candidateRev] = append(trackedBranchTips[candidateRev], candidate)
	}

	// Get history of the branch we're tracking
	history, err := e.git.GetCommitHistorySHAs(branchName)
	if err != nil {
		return nil, err
	}

	// Iterate through history (newest to oldest) and find the first tracked tip(s)
	for i := 0; i < len(history); i++ {
		sha := history[i]
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
		commits, err := e.GetAllCommitsInternal(branchName, CommitFormatSHA)
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

// GetAllCommitsInternal returns commits for a branch in various formats
func (e *engineImpl) GetAllCommitsInternal(branchName string, format CommitFormat) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if branch is trunk
	if branchName == e.trunk {
		// Trunk is the base, so it has no commits "on" it relative to a parent
		return []string{}, nil
	}

	// Get metadata to find parent revision
	meta, err := e.readMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	// Get branch revision
	branchRevision, err := e.GetRevisionInternal(branchName)
	if err != nil {
		return nil, err
	}

	// Get parent revision (base)
	var baseRevision string
	if meta.ParentBranchRevision != nil {
		baseRevision = *meta.ParentBranchRevision
	}

	// Get SHAs first
	shas, err := e.git.GetCommitRangeSHAs(baseRevision, branchRevision)
	if err != nil {
		return nil, err
	}

	if format == CommitFormatSHA {
		return shas, nil
	}

	// Format results
	result := make([]string, len(shas))
	for i, sha := range shas {
		var formatted string
		switch format {
		case CommitFormatSubject:
			formatted, _ = e.git.RunGitCommand("log", "-1", "--format=%s", sha)
		case CommitFormatMessage:
			formatted, _ = e.git.RunGitCommand("log", "-1", "--format=%B", sha)
		case CommitFormatReadable:
			formatted, _ = e.git.RunGitCommand("log", "-1", "--format=%h %s", sha)
		default:
			return nil, fmt.Errorf("unknown commit format: %s", format)
		}
		result[i] = strings.TrimSpace(formatted)
	}

	return result, nil
}

// GetRelativeStackUpstack returns all branches in the upstack (descendants)
func (e *engineImpl) GetRelativeStackUpstack(branch Branch) []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.getRelativeStackUpstackInternal(branch.GetName())
}

// GetRelativeStackDownstack returns all branches in the downstack (ancestors)
func (e *engineImpl) GetRelativeStackDownstack(branch Branch) []Branch {
	rng := StackRange{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	return e.GetRelativeStackInternal(branch.GetName(), rng)
}

// GetFullStack returns the entire stack containing the branch
func (e *engineImpl) GetFullStack(branch Branch) []Branch {
	rng := StackRange{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: true,
	}
	return e.GetRelativeStackInternal(branch.GetName(), rng)
}

// SortBranchesTopologically sorts branches so parents come before children.
// This ensures correct restack order (bottom of stack first).
func (e *engineImpl) SortBranchesTopologically(branches []Branch) []Branch {
	if len(branches) == 0 {
		return branches
	}

	// Calculate depth for each branch (distance from trunk)
	depths := make(map[string]int)
	var getDepth func(branchName string) int
	getDepth = func(branchName string) int {
		if depth, ok := depths[branchName]; ok {
			return depth
		}

		e.mu.RLock()
		isTrunk := branchName == e.trunk
		parent := e.parentMap[branchName]
		e.mu.RUnlock()

		if isTrunk {
			depths[branchName] = 0
			return 0
		}
		if parent == "" || e.IsTrunkInternal(parent) {
			depths[branchName] = 1
			return 1
		}
		depths[branchName] = getDepth(parent) + 1
		return depths[branchName]
	}

	// Calculate depth for all branches
	for _, branch := range branches {
		getDepth(branch.GetName())
	}

	// Sort by depth (parents first, then children)
	result := make([]Branch, len(branches))
	copy(result, branches)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if depths[result[i].GetName()] > depths[result[j].GetName()] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetDeletionStatus checks if a branch should be deleted
func (e *engineImpl) GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error) {
	// Check PR info
	branch := e.GetBranch(branchName)
	prInfo, err := e.GetPrInfo(branch)
	if err == nil && prInfo != nil {
		const (
			prStateClosed = "CLOSED"
			prStateMerged = "MERGED"
		)
		if prInfo.State() == prStateClosed {
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is closed on GitHub", branchName)}, nil
		}
		if prInfo.State() == prStateMerged {
			base := prInfo.Base()
			if base == "" {
				base = e.Trunk().GetName()
			}
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is merged into %s", branchName, base)}, nil
		}
	}

	// Check if merged into trunk
	merged, err := e.IsMergedIntoTrunk(ctx, branchName)
	if err == nil && merged {
		return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is merged into %s", branchName, e.Trunk().GetName())}, nil
	}

	// Check if empty
	empty, err := e.IsBranchEmpty(ctx, branchName)
	if err == nil && empty {
		// Only delete empty branches if they have a PR
		if prInfo != nil && prInfo.Number() != nil {
			return DeletionStatus{SafeToDelete: true, Reason: fmt.Sprintf("%s is empty", branchName)}, nil
		}
	}

	return DeletionStatus{SafeToDelete: false, Reason: ""}, nil
}

// BranchesDepthFirst returns an iterator that yields branches starting from startBranch in depth-first order.
// Each iteration yields (branchName, depth) where depth is 0 for the start branch.
// The iterator can be used with range loops and supports early termination with break.
func (e *engineImpl) BranchesDepthFirst(startBranch Branch) iter.Seq2[Branch, int] {
	return func(yield func(Branch, int) bool) {
		visited := make(map[string]bool)
		var visit func(branch string, depth int) bool
		visit = func(branch string, depth int) bool {
			if visited[branch] {
				return true // cycle detection
			}
			visited[branch] = true

			if !yield(NewBranch(branch, e), depth) {
				return false // iterator wants to stop
			}

			children := e.GetChildrenInternal(branch)
			for _, child := range children {
				if !visit(child.GetName(), depth+1) {
					return false
				}
			}
			return true
		}

		visit(startBranch.GetName(), 0)
	}
}

// GetRemote returns the default remote name
func (e *engineImpl) GetRemote() string {
	return e.git.GetRemote()
}

// GetBranchRemoteDifference returns a string describing the difference between local and remote branch
func (e *engineImpl) GetBranchRemoteDifference(branchName string) (string, error) {
	localSha, err := e.git.GetRevision(branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get local SHA for %s: %w", branchName, err)
	}

	remoteSha, err := e.git.GetRemoteRevision(branchName)
	if err != nil {
		remote := e.git.GetRemote()
		remoteShas, err := e.git.FetchRemoteShas(remote)
		if err != nil {
			localShort := localSha
			if len(localSha) > 7 {
				localShort = localSha[:7]
			}
			return fmt.Sprintf("local: %s (unable to fetch remote SHA)", localShort), nil
		}
		var exists bool
		remoteSha, exists = remoteShas[branchName]
		if !exists {
			localShort := localSha
			if len(localSha) > 7 {
				localShort = localSha[:7]
			}
			return fmt.Sprintf("local: %s (branch not found on remote)", localShort), nil
		}
	}

	if localSha == remoteSha {
		return "", nil
	}

	localShort := localSha
	if len(localSha) > 7 {
		localShort = localSha[:7]
	}
	remoteShort := remoteSha
	if len(remoteSha) > 7 {
		remoteShort = remoteSha[:7]
	}

	remote := e.git.GetRemote()
	remoteBranchRef := "refs/remotes/" + remote + "/" + branchName
	commonAncestor, err := e.git.GetMergeBaseByRef(branchName, remoteBranchRef)
	if err != nil {
		return fmt.Sprintf("local: %s, remote: %s (likely local is ahead)", localShort, remoteShort), nil //nolint:nilerr
	}

	switch {
	case commonAncestor == localSha:
		return fmt.Sprintf("local is behind remote (local: %s, remote: %s)", localShort, remoteShort), nil
	case commonAncestor == remoteSha:
		return fmt.Sprintf("local is ahead of remote (local: %s, remote: %s)", localShort, remoteShort), nil
	default:
		return fmt.Sprintf("local and remote have diverged (local: %s, remote: %s)", localShort, remoteShort), nil
	}
}

// HasStagedChanges checks if there are staged changes in the repository
func (e *engineImpl) HasStagedChanges(ctx context.Context) (bool, error) {
	return e.git.HasStagedChanges(ctx)
}

// HasUnstagedChanges checks if there are unstaged changes in the repository
func (e *engineImpl) HasUnstagedChanges(ctx context.Context) (bool, error) {
	return e.git.HasUnstagedChanges(ctx)
}

// GetMergeBase returns the merge base between two revisions
func (e *engineImpl) GetMergeBase(rev1, rev2 string) (string, error) {
	return e.git.GetMergeBase(rev1, rev2)
}

// GetChangedFiles returns the list of files changed between base and head
func (e *engineImpl) GetChangedFiles(ctx context.Context, base, head string) ([]string, error) {
	return e.git.GetChangedFiles(ctx, base, head)
}

// ListWorktrees returns a list of worktree paths
func (e *engineImpl) ListWorktrees(ctx context.Context) ([]string, error) {
	return e.git.ListWorktrees(ctx)
}

// GetWorkingDir returns the current working directory
func (e *engineImpl) GetWorkingDir() string {
	return e.git.GetWorkingDir()
}

// RunGitCommandWithContext runs a git command with context
func (e *engineImpl) RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	return e.git.RunGitCommandWithContext(ctx, args...)
}

// RunGitCommandRawWithContext runs a git command raw with context
func (e *engineImpl) RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error) {
	return e.git.RunGitCommandRawWithContext(ctx, args...)
}

// ParseStagedHunks parses the output of `git diff --cached` into structured hunks
func (e *engineImpl) ParseStagedHunks(ctx context.Context) ([]git.Hunk, error) {
	return e.git.ParseStagedHunks(ctx)
}

// ShowDiff returns the diff between two refs with optional stat mode
func (e *engineImpl) ShowDiff(ctx context.Context, left, right string, stat bool) (string, error) {
	return e.git.ShowDiff(ctx, left, right, stat)
}

// ShowCommits returns commit log with optional patches/stat
func (e *engineImpl) ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error) {
	return e.git.ShowCommits(ctx, base, head, patch, stat)
}

// GetUnmergedFiles returns list of files with merge conflicts
func (e *engineImpl) GetUnmergedFiles(ctx context.Context) ([]string, error) {
	return e.git.GetUnmergedFiles(ctx)
}

// GetParentCommitSHA returns the parent commit SHA of a commit
func (e *engineImpl) GetParentCommitSHA(commitSHA string) (string, error) {
	return e.git.GetParentCommitSHA(commitSHA)
}

// GetCommitSHA returns the SHA at a relative position (0 = HEAD, 1 = HEAD~1)
func (e *engineImpl) GetCommitSHA(branchName string, offset int) (string, error) {
	return e.git.GetCommitSHA(branchName, offset)
}

// IsAncestor checks if one commit is an ancestor of another
func (e *engineImpl) IsAncestor(ancestor, descendant string) (bool, error) {
	return e.git.IsAncestor(ancestor, descendant)
}
