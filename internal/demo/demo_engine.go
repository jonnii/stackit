package demo

import (
	"context"
	"fmt"
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

// DemoEngine implements the engine.Engine interface with simulated data
type DemoEngine struct {
	parentMap   map[string]string
	childrenMap map[string][]string
	prInfoMap   map[string]*engine.PrInfo
}

// NewDemoEngine creates a new demo engine with simulated stack data
func NewDemoEngine() *DemoEngine {
	e := &DemoEngine{
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

func (e *DemoEngine) AllBranchNames() []string {
	names := []string{GetDemoTrunk()}
	for _, b := range GetDemoBranches() {
		names = append(names, b.Name)
	}
	return names
}

func (e *DemoEngine) CurrentBranch() string {
	return GetDemoCurrentBranch()
}

func (e *DemoEngine) Trunk() string {
	return GetDemoTrunk()
}

func (e *DemoEngine) GetParent(branchName string) string {
	return e.parentMap[branchName]
}

func (e *DemoEngine) GetParentPrecondition(branchName string) string {
	parent := e.parentMap[branchName]
	if parent == "" {
		panic(fmt.Sprintf("branch %s has no parent", branchName))
	}
	return parent
}

func (e *DemoEngine) GetChildren(branchName string) []string {
	return e.childrenMap[branchName]
}

func (e *DemoEngine) GetRelativeStack(branchName string, scope engine.Scope) []string {
	var result []string

	// Add ancestors if RecursiveParents
	if scope.RecursiveParents {
		ancestors := []string{}
		current := branchName
		for {
			parent := e.parentMap[current]
			if parent == "" || parent == GetDemoTrunk() {
				break
			}
			ancestors = append([]string{parent}, ancestors...)
			current = parent
		}
		result = append(result, ancestors...)
	}

	// Add current branch
	if scope.IncludeCurrent {
		result = append(result, branchName)
	}

	// Add descendants if RecursiveChildren
	if scope.RecursiveChildren {
		result = append(result, e.getDescendants(branchName)...)
	}

	return result
}

func (e *DemoEngine) getDescendants(branchName string) []string {
	children := e.childrenMap[branchName]
	result := make([]string, 0, len(children))
	for _, child := range children {
		result = append(result, child)
		result = append(result, e.getDescendants(child)...)
	}
	return result
}

// IsTrunk returns true if the branch is trunk in the demo engine
func (e *DemoEngine) IsTrunk(branchName string) bool {
	return branchName == GetDemoTrunk()
}

// IsBranchTracked returns true if the branch is tracked in the demo engine
func (e *DemoEngine) IsBranchTracked(branchName string) bool {
	if branchName == GetDemoTrunk() {
		return true
	}
	_, exists := e.parentMap[branchName]
	return exists
}

// IsBranchFixed returns true in the demo engine
func (e *DemoEngine) IsBranchFixed(_ context.Context, _ string) bool {
	return true // All demo branches are "fixed" (not needing restack)
}

// GetCommitDate returns a fake commit date in the demo engine
func (e *DemoEngine) GetCommitDate(_ context.Context, _ string) (time.Time, error) {
	return time.Now().Add(-24 * time.Hour), nil
}

// GetCommitAuthor returns a fake commit author in the demo engine
func (e *DemoEngine) GetCommitAuthor(_ context.Context, _ string) (string, error) {
	return "Demo User <demo@example.com>", nil
}

// GetRevision returns a fake revision in the demo engine
func (e *DemoEngine) GetRevision(_ context.Context, branchName string) (string, error) {
	// Return fake SHA based on branch name
	return fmt.Sprintf("abc%x123", len(branchName)), nil
}

// FindBranchForCommit returns the current branch in the demo engine
func (e *DemoEngine) FindBranchForCommit(_ context.Context, _ string) (string, error) {
	// For demo, just return the current branch
	return e.CurrentBranch(), nil
}

// GetRelativeStackUpstack returns descendants in the demo engine
func (e *DemoEngine) GetRelativeStackUpstack(branchName string) []string {
	return e.getDescendants(branchName)
}

// IsMergedIntoTrunk returns false in the demo engine
func (e *DemoEngine) IsMergedIntoTrunk(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// IsBranchEmpty returns false in the demo engine
func (e *DemoEngine) IsBranchEmpty(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// FindMostRecentTrackedAncestors returns the parent in the demo engine
func (e *DemoEngine) FindMostRecentTrackedAncestors(_ context.Context, branchName string) ([]string, error) {
	parent := e.parentMap[branchName]
	if parent == "" {
		return nil, fmt.Errorf("no tracked ancestor found for branch %s", branchName)
	}
	return []string{parent}, nil
}

// BranchWriter interface implementation

// TrackBranch tracks a branch in the demo engine
func (e *DemoEngine) TrackBranch(_ context.Context, branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)
	return nil
}

// UntrackBranch untracks a branch in the demo engine
func (e *DemoEngine) UntrackBranch(_ context.Context, branchName string) error {
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
func (e *DemoEngine) SetParent(_ context.Context, branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	return nil
}

// DeleteBranch deletes a branch in the demo engine
func (e *DemoEngine) DeleteBranch(_ context.Context, branchName string) error {
	delete(e.parentMap, branchName)
	delete(e.prInfoMap, branchName)
	return nil
}

// Reset resets the demo engine
func (e *DemoEngine) Reset(_ context.Context, _ string) error {
	return nil
}

// Rebuild rebuilds the demo engine
func (e *DemoEngine) Rebuild(_ context.Context, _ string) error {
	return nil
}

// PRManager interface implementation

// GetPrInfo returns PR info in the demo engine
func (e *DemoEngine) GetPrInfo(_ context.Context, branchName string) (*engine.PrInfo, error) {
	if info, exists := e.prInfoMap[branchName]; exists {
		return info, nil
	}
	return nil, nil
}

// UpsertPrInfo upserts PR info in the demo engine
func (e *DemoEngine) UpsertPrInfo(_ context.Context, branchName string, prInfo *engine.PrInfo) error {
	simulateDelay(delayLong) // GitHub API call to create/update PR
	e.prInfoMap[branchName] = prInfo
	return nil
}

// SyncManager interface implementation

// BranchMatchesRemote returns false in the demo engine
func (e *DemoEngine) BranchMatchesRemote(_ context.Context, _ string) (bool, error) {
	return false, nil // Demo branches never match remote (so submit always has work to do)
}

// PopulateRemoteShas simulates fetching remote refs in the demo engine
func (e *DemoEngine) PopulateRemoteShas(_ context.Context) error {
	simulateDelay(delayMedium) // Fetching remote refs takes time
	return nil
}

// PushBranch simulates git push in the demo engine
func (e *DemoEngine) PushBranch(_ context.Context, _ string, _ string, _ bool, _ bool) error {
	simulateDelay(delayMedium) // Git push takes time
	return nil
}

// PullTrunk simulates git pull in the demo engine
func (e *DemoEngine) PullTrunk(_ context.Context) (engine.PullResult, error) {
	simulateDelay(delayLong) // Git pull takes time
	return engine.PullUnneeded, nil
}

// ResetTrunkToRemote simulates resetting trunk to remote in the demo engine
func (e *DemoEngine) ResetTrunkToRemote(_ context.Context) error {
	return nil
}

// RestackBranch simulates restack in the demo engine
func (e *DemoEngine) RestackBranch(_ context.Context, _ string) (engine.RestackBranchResult, error) {
	simulateDelay(delayMedium) // Rebase operation takes time
	return engine.RestackBranchResult{
		Result: engine.RestackUnneeded,
	}, nil
}

// ContinueRebase simulates continuing rebase in the demo engine
func (e *DemoEngine) ContinueRebase(_ context.Context, _ string) (engine.ContinueRebaseResult, error) {
	return engine.ContinueRebaseResult{
		Result:     0, // RebaseDone
		BranchName: GetDemoCurrentBranch(),
	}, nil
}

// SquashManager interface implementation

// SquashCurrentBranch simulates squash in the demo engine
func (e *DemoEngine) SquashCurrentBranch(_ context.Context, _ engine.SquashOptions) error {
	simulateDelay(delayMedium) // Squash takes time
	return nil
}

// SplitManager interface implementation

// GetAllCommits returns fake commits in the demo engine
func (e *DemoEngine) GetAllCommits(_ context.Context, branchName string, _ engine.CommitFormat) ([]string, error) {
	return []string{
		"abc1234 Initial commit for " + branchName,
		"def5678 Add feature implementation",
	}, nil
}

// ApplySplitToCommits simulates split in the demo engine
func (e *DemoEngine) ApplySplitToCommits(_ context.Context, _ engine.ApplySplitOptions) error {
	simulateDelay(delayLong) // Split involves multiple git operations
	return nil
}

// Detach simulates detach in the demo engine
func (e *DemoEngine) Detach(_ context.Context, _ string) error {
	return nil
}

// DetachAndResetBranchChanges simulates detach and reset in the demo engine
func (e *DemoEngine) DetachAndResetBranchChanges(_ context.Context, _ string) error {
	return nil
}

// ForceCheckoutBranch simulates force checkout in the demo engine
func (e *DemoEngine) ForceCheckoutBranch(_ context.Context, _ string) error {
	simulateDelay(delayShort) // Checkout is fast
	return nil
}

// Ensure DemoEngine implements engine.Engine
var _ engine.Engine = (*DemoEngine)(nil)
