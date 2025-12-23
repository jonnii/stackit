package demo

import (
	"context"
	"fmt"
	"iter"
	"os"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// Delay constants for simulating real operations
const (
	delayShort  = 150 * time.Millisecond
	delayMedium = 300 * time.Millisecond
	delayLong   = 500 * time.Millisecond
)

// simulateDelay adds a random delay around the base duration
func simulateDelay(base time.Duration) {
	// Use a fixed jitter for demo to avoid weak random number generator warnings
	// and because true randomness isn't critical for demo simulation
	jitter := time.Duration(base.Nanoseconds()%100) * time.Millisecond
	time.Sleep(base + jitter)
}

// IsDemoMode returns true if STACKIT_DEMO environment variable is set
func IsDemoMode() bool {
	return os.Getenv("STACKIT_DEMO") != ""
}

func init() {
	// Register the demo engine factory with runtime package
	runtime.DemoEngineFactory = func() engine.Engine {
		return NewDemoEngine()
	}
}

// NewDemoContext creates a demo engine and context if in demo mode.
// Returns (context, true) if in demo mode, (nil, false) otherwise.
// Deprecated: Use runtime.NewContextAuto instead.
func NewDemoContext() (*runtime.Context, bool) {
	if !IsDemoMode() {
		return nil, false
	}
	eng := NewDemoEngine()
	return runtime.NewContext(eng), true
}

// Engine implements the engine.Engine interface with simulated data
type Engine struct {
	parentMap   map[string]string
	childrenMap map[string][]string
	prInfoMap   map[string]*engine.PrInfo
}

// NewDemoEngine creates a new demo engine with simulated stack data
func NewDemoEngine() *Engine {
	e := &Engine{
		parentMap:   make(map[string]string),
		childrenMap: make(map[string][]string),
		prInfoMap:   make(map[string]*engine.PrInfo),
	}

	// Build parent/children maps from demo data
	for _, b := range GetDemoBranches() {
		e.parentMap[b.Name] = b.Parent
		e.childrenMap[b.Parent] = append(e.childrenMap[b.Parent], b.Name)

		// Build PR info
		num := b.PRNumber
		e.prInfoMap[b.Name] = &engine.PrInfo{
			Number:  &num,
			Title:   b.PRTitle,
			Body:    "Demo PR body for " + b.Name,
			IsDraft: b.IsDraft,
			State:   b.PRState,
			Base:    b.Parent,
			URL:     fmt.Sprintf("https://github.com/example/repo/pull/%d", num),
		}
	}

	return e
}

// BranchReader interface implementation

// AllBranches returns simulated branches
func (e *Engine) AllBranches() []engine.Branch {
	names := []string{GetDemoTrunk()}
	for _, b := range GetDemoBranches() {
		names = append(names, b.Name)
	}
	branches := make([]engine.Branch, len(names))
	for i, name := range names {
		branches[i] = engine.Branch{Name: name, Reader: e}
	}
	return branches
}

// CurrentBranch returns the simulated current branch
func (e *Engine) CurrentBranch() *engine.Branch {
	currentBranchName := GetDemoCurrentBranch()
	if currentBranchName == "" {
		return nil
	}
	return &engine.Branch{Name: currentBranchName, Reader: e}
}

// Trunk returns the simulated trunk branch
func (e *Engine) Trunk() engine.Branch {
	return engine.Branch{Name: GetDemoTrunk(), Reader: e}
}

// GetParent returns the simulated parent of a branch
func (e *Engine) GetParent(branchName string) *engine.Branch {
	parentName := e.parentMap[branchName]
	if parentName == "" {
		return nil
	}
	return &engine.Branch{Name: parentName, Reader: e}
}

// GetChildrenInternal returns the simulated children of a branch (internal method for Branch type)
func (e *Engine) GetChildrenInternal(branchName string) []engine.Branch {
	children := e.childrenMap[branchName]
	branches := make([]engine.Branch, len(children))
	for i, name := range children {
		branches[i] = engine.Branch{Name: name, Reader: e}
	}
	return branches
}

// GetRelativeStack returns branches in stack order
func (e *Engine) GetRelativeStack(branch engine.Branch, scope engine.Scope) []engine.Branch {
	var result []engine.Branch

	// Add ancestors if RecursiveParents
	if scope.RecursiveParents {
		ancestors := []engine.Branch{}
		current := branch.Name
		for {
			parent := e.parentMap[current]
			if parent == "" || parent == GetDemoTrunk() {
				break
			}
			ancestors = append([]engine.Branch{{Name: parent, Reader: e}}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch
	if scope.IncludeCurrent {
		result = append(result, branch)
	}

	// Add descendants if RecursiveChildren
	if scope.RecursiveChildren {
		descendants := e.getDescendants(branch.Name)
		for _, name := range descendants {
			result = append(result, engine.Branch{Name: name, Reader: e})
		}
	}

	return result
}

// GetRelativeStackInternal returns branches in stack order (internal method used by Branch type)
func (e *Engine) GetRelativeStackInternal(branchName string, scope engine.Scope) []engine.Branch {
	var result []engine.Branch

	// Add ancestors if RecursiveParents
	if scope.RecursiveParents {
		ancestors := []engine.Branch{}
		current := branchName
		for {
			parent := e.parentMap[current]
			if parent == "" || parent == GetDemoTrunk() {
				break
			}
			ancestors = append([]engine.Branch{{Name: parent, Reader: e}}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch
	if scope.IncludeCurrent {
		result = append(result, engine.Branch{Name: branchName, Reader: e})
	}

	// Add descendants if RecursiveChildren
	if scope.RecursiveChildren {
		descendants := e.getDescendants(branchName)
		for _, name := range descendants {
			result = append(result, engine.Branch{Name: name, Reader: e})
		}
	}

	return result
}

func (e *Engine) getDescendants(branchName string) []string {
	children := e.childrenMap[branchName]
	result := make([]string, 0, len(children))
	for _, child := range children {
		result = append(result, child)
		result = append(result, e.getDescendants(child)...)
	}
	return result
}

// GetBranch returns a Branch wrapper for the given branch name
func (e *Engine) GetBranch(branchName string) engine.Branch {
	return engine.Branch{Name: branchName, Reader: e}
}

// IsTrunkInternal returns true if the branch is trunk in the demo engine (internal method used by Branch type)
func (e *Engine) IsTrunkInternal(branchName string) bool {
	return branchName == GetDemoTrunk()
}

// IsBranchTrackedInternal returns true if the branch is tracked in the demo engine (internal method used by Branch type)
func (e *Engine) IsBranchTrackedInternal(branchName string) bool {
	if branchName == GetDemoTrunk() {
		return true
	}
	_, exists := e.parentMap[branchName]
	return exists
}

// IsBranchUpToDateInternal returns true in the demo engine
func (e *Engine) IsBranchUpToDateInternal(_ string) bool {
	return true // All demo branches are "fixed" (not needing restack)
}

// GetCommitDateInternal returns a fake commit date in the demo engine
func (e *Engine) GetCommitDateInternal(_ string) (time.Time, error) {
	return time.Now().Add(-24 * time.Hour), nil
}

// GetCommitAuthorInternal returns a fake commit author in the demo engine
func (e *Engine) GetCommitAuthorInternal(_ string) (string, error) {
	return "Demo User <demo@example.com>", nil
}

// GetRevisionInternal returns a fake revision in the demo engine
func (e *Engine) GetRevisionInternal(branchName string) (string, error) {
	// Return fake SHA based on branch name
	return fmt.Sprintf("abc%x123", len(branchName)), nil
}

// FindBranchForCommit returns the current branch in the demo engine
func (e *Engine) FindBranchForCommit(_ string) (string, error) {
	// For demo, just return the current branch
	currentBranch := e.CurrentBranch()
	if currentBranch == nil {
		return "", fmt.Errorf("not on a branch")
	}
	return currentBranch.Name, nil
}

// GetRelativeStackUpstack returns descendants in the demo engine
func (e *Engine) GetRelativeStackUpstack(branchName string) []engine.Branch {
	descendants := e.getDescendants(branchName)
	result := make([]engine.Branch, len(descendants))
	for i, name := range descendants {
		result[i] = engine.Branch{Name: name, Reader: e}
	}
	return result
}

// GetRelativeStackDownstack returns ancestors in the demo engine
func (e *Engine) GetRelativeStackDownstack(branchName string) []engine.Branch {
	branch := e.GetBranch(branchName)
	return e.GetRelativeStack(branch, engine.Scope{RecursiveParents: true, IncludeCurrent: false, RecursiveChildren: false})
}

// GetFullStack returns the entire stack in the demo engine
func (e *Engine) GetFullStack(branchName string) []engine.Branch {
	branch := e.GetBranch(branchName)
	return e.GetRelativeStack(branch, engine.Scope{RecursiveParents: true, IncludeCurrent: true, RecursiveChildren: true})
}

// SortBranchesTopologically simulates topological sort in the demo engine
func (e *Engine) SortBranchesTopologically(branches []string) []string {
	// For demo, just return the branches as-is or do a simple sort if needed
	return branches
}

// IsMergedIntoTrunk returns false in the demo engine
func (e *Engine) IsMergedIntoTrunk(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// IsBranchEmpty returns false in the demo engine
func (e *Engine) IsBranchEmpty(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// GetDeletionStatus checks if a branch can be deleted in the demo engine
func (e *Engine) GetDeletionStatus(_ context.Context, branchName string) (engine.DeletionStatus, error) {
	// In demo mode, some branches are simulated as merged
	for _, b := range GetDemoBranches() {
		if b.Name == branchName && b.PRState == "MERGED" {
			return engine.DeletionStatus{SafeToDelete: true, Reason: "merged in demo"}, nil
		}
	}
	return engine.DeletionStatus{SafeToDelete: false}, nil
}

// FindMostRecentTrackedAncestors returns the parent in the demo engine
func (e *Engine) FindMostRecentTrackedAncestors(_ context.Context, branchName string) ([]string, error) {
	parent := e.parentMap[branchName]
	if parent == "" {
		return nil, fmt.Errorf("no tracked ancestor found for branch %s", branchName)
	}
	return []string{parent}, nil
}

// BranchesDepthFirst returns an iterator that yields branches starting from startBranch in depth-first order.
// Each iteration yields (branch, depth) where depth is 0 for the start branch.
// The iterator can be used with range loops and supports early termination with break.
func (e *Engine) BranchesDepthFirst(startBranch string) iter.Seq2[engine.Branch, int] {
	return func(yield func(engine.Branch, int) bool) {
		visited := make(map[string]bool)
		var visit func(branch string, depth int) bool
		visit = func(branch string, depth int) bool {
			if visited[branch] {
				return true // cycle detection
			}
			visited[branch] = true

			if !yield(engine.Branch{Name: branch, Reader: e}, depth) {
				return false // iterator wants to stop
			}

			children := e.GetChildrenInternal(branch)
			for _, child := range children {
				if !visit(child.Name, depth+1) {
					return false
				}
			}
			return true
		}

		visit(startBranch, 0)
	}
}

// BranchWriter interface implementation

// TrackBranch tracks a branch in the demo engine
func (e *Engine) TrackBranch(_ context.Context, branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)
	return nil
}

// UntrackBranch untracks a branch in the demo engine
func (e *Engine) UntrackBranch(branchName string) error {
	delete(e.parentMap, branchName)
	// Rebuild children map (simplified for demo)
	newChildrenMap := make(map[string][]string)
	for branch, parent := range e.parentMap {
		newChildrenMap[parent] = append(newChildrenMap[parent], branch)
	}
	e.childrenMap = newChildrenMap
	return nil
}

// SetParent sets the parent of a branch in the demo engine
func (e *Engine) SetParent(_ context.Context, branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	return nil
}

// DeleteBranch deletes a branch in the demo engine
func (e *Engine) DeleteBranch(_ context.Context, branchName string) error {
	delete(e.parentMap, branchName)
	delete(e.prInfoMap, branchName)
	return nil
}

// DeleteBranches deletes multiple branches in the demo engine
func (e *Engine) DeleteBranches(ctx context.Context, branchNames []string) ([]string, error) {
	for _, b := range branchNames {
		_ = e.DeleteBranch(ctx, b)
	}
	return nil, nil
}

// Reset resets the demo engine
func (e *Engine) Reset(_ string) error {
	return nil
}

// Rebuild rebuilds the demo engine
func (e *Engine) Rebuild(_ string) error {
	return nil
}

// PRManager interface implementation

// GetPrInfo returns PR info in the demo engine
func (e *Engine) GetPrInfo(branchName string) (*engine.PrInfo, error) {
	if info, exists := e.prInfoMap[branchName]; exists {
		return info, nil
	}
	return nil, nil
}

// UpsertPrInfo upserts PR info in the demo engine
func (e *Engine) UpsertPrInfo(branchName string, prInfo *engine.PrInfo) error {
	simulateDelay(delayLong) // GitHub API call to create/update PR
	e.prInfoMap[branchName] = prInfo
	return nil
}

// GetPRSubmissionStatus returns the submission status in the demo engine
func (e *Engine) GetPRSubmissionStatus(branchName string) (engine.PRSubmissionStatus, error) {
	info, _ := e.GetPrInfo(branchName)
	if info == nil {
		return engine.PRSubmissionStatus{Action: "create", NeedsUpdate: true}, nil
	}
	return engine.PRSubmissionStatus{Action: "update", NeedsUpdate: true, PRNumber: info.Number, PRInfo: info}, nil
}

// SyncManager interface implementation

// BranchMatchesRemote returns false in the demo engine
func (e *Engine) BranchMatchesRemote(_ string) (bool, error) {
	return false, nil // Demo branches never match remote (so submit always has work to do)
}

// PopulateRemoteShas simulates fetching remote refs in the demo engine
func (e *Engine) PopulateRemoteShas() error {
	simulateDelay(delayMedium) // Fetching remote refs takes time
	return nil
}

// PushBranch simulates git push in the demo engine
func (e *Engine) PushBranch(_ context.Context, _ string, _ string, _ bool, _ bool) error {
	simulateDelay(delayMedium) // Git push takes time
	return nil
}

// PullTrunk simulates git pull in the demo engine
func (e *Engine) PullTrunk(_ context.Context) (engine.PullResult, error) {
	simulateDelay(delayLong) // Git pull takes time
	return engine.PullUnneeded, nil
}

// ResetTrunkToRemote simulates resetting trunk to remote in the demo engine
func (e *Engine) ResetTrunkToRemote(_ context.Context) error {
	return nil
}

// RestackBranch simulates restack in the demo engine
func (e *Engine) RestackBranch(_ context.Context, _ string) (engine.RestackBranchResult, error) {
	simulateDelay(delayMedium) // Rebase operation takes time
	return engine.RestackBranchResult{
		Result: engine.RestackUnneeded,
	}, nil
}

// RestackBranches simulates batch restack in the demo engine
func (e *Engine) RestackBranches(ctx context.Context, branchNames []string) (engine.RestackBatchResult, error) {
	results := make(map[string]engine.RestackBranchResult)
	for _, b := range branchNames {
		res, _ := e.RestackBranch(ctx, b)
		results[b] = res
	}
	return engine.RestackBatchResult{Results: results}, nil
}

// ContinueRebase simulates continuing rebase in the demo engine
func (e *Engine) ContinueRebase(_ context.Context, _ string) (engine.ContinueRebaseResult, error) {
	return engine.ContinueRebaseResult{
		Result:     0, // RebaseDone
		BranchName: GetDemoCurrentBranch(),
	}, nil
}

// SquashManager interface implementation

// SquashCurrentBranch simulates squash in the demo engine
func (e *Engine) SquashCurrentBranch(_ context.Context, _ engine.SquashOptions) error {
	simulateDelay(delayMedium) // Squash takes time
	return nil
}

// SplitManager interface implementation

// GetAllCommitsInternal returns fake commits in the demo engine
func (e *Engine) GetAllCommitsInternal(branchName string, _ engine.CommitFormat) ([]string, error) {
	return []string{
		"abc1234 Initial commit for " + branchName,
		"def5678 Add feature implementation",
	}, nil
}

// ApplySplitToCommits simulates split in the demo engine
func (e *Engine) ApplySplitToCommits(_ context.Context, _ engine.ApplySplitOptions) error {
	simulateDelay(delayLong) // Split involves multiple git operations
	return nil
}

// Detach simulates detach in the demo engine
func (e *Engine) Detach(_ context.Context, _ string) error {
	return nil
}

// DetachAndResetBranchChanges simulates detach and reset in the demo engine
func (e *Engine) DetachAndResetBranchChanges(_ context.Context, _ string) error {
	return nil
}

// ForceCheckoutBranch simulates force checkout in the demo engine
func (e *Engine) ForceCheckoutBranch(_ context.Context, _ string) error {
	simulateDelay(delayShort) // Checkout is fast
	return nil
}

// UndoManager interface implementation

// TakeSnapshot simulates taking a snapshot in the demo engine
func (e *Engine) TakeSnapshot(_ string, _ []string) error {
	// In demo mode, snapshots are not persisted
	return nil
}

// GetSnapshots returns empty snapshots list in the demo engine
func (e *Engine) GetSnapshots() ([]engine.SnapshotInfo, error) {
	// Demo mode doesn't support undo
	return []engine.SnapshotInfo{}, nil
}

// LoadSnapshot returns an error in the demo engine
func (e *Engine) LoadSnapshot(_ string) (*engine.Snapshot, error) {
	return nil, fmt.Errorf("undo not supported in demo mode")
}

// RestoreSnapshot returns an error in the demo engine
func (e *Engine) RestoreSnapshot(_ context.Context, _ string) error {
	return fmt.Errorf("undo not supported in demo mode")
}

// Ensure Engine implements engine.Engine
var _ engine.Engine = (*Engine)(nil)
