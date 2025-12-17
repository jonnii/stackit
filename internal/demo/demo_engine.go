package demo

import (
	"fmt"
	"time"

	"stackit.dev/stackit/internal/engine"
)

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
	var result []string
	children := e.childrenMap[branchName]
	for _, child := range children {
		result = append(result, child)
		result = append(result, e.getDescendants(child)...)
	}
	return result
}

func (e *DemoEngine) IsTrunk(branchName string) bool {
	return branchName == GetDemoTrunk()
}

func (e *DemoEngine) IsBranchTracked(branchName string) bool {
	if branchName == GetDemoTrunk() {
		return true
	}
	_, exists := e.parentMap[branchName]
	return exists
}

func (e *DemoEngine) IsBranchFixed(branchName string) bool {
	return true // All demo branches are "fixed" (not needing restack)
}

func (e *DemoEngine) GetCommitDate(branchName string) (time.Time, error) {
	return time.Now().Add(-24 * time.Hour), nil
}

func (e *DemoEngine) GetCommitAuthor(branchName string) (string, error) {
	return "Demo User <demo@example.com>", nil
}

func (e *DemoEngine) GetRevision(branchName string) (string, error) {
	// Return fake SHA based on branch name
	return fmt.Sprintf("abc%x123", len(branchName)), nil
}

func (e *DemoEngine) GetRelativeStackUpstack(branchName string) []string {
	return e.getDescendants(branchName)
}

func (e *DemoEngine) IsMergedIntoTrunk(branchName string) (bool, error) {
	return false, nil
}

func (e *DemoEngine) IsBranchEmpty(branchName string) (bool, error) {
	return false, nil
}

// BranchWriter interface implementation

func (e *DemoEngine) TrackBranch(branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)
	return nil
}

func (e *DemoEngine) SetParent(branchName string, parentBranchName string) error {
	e.parentMap[branchName] = parentBranchName
	return nil
}

func (e *DemoEngine) DeleteBranch(branchName string) error {
	delete(e.parentMap, branchName)
	delete(e.prInfoMap, branchName)
	return nil
}

func (e *DemoEngine) Reset(newTrunkName string) error {
	return nil
}

func (e *DemoEngine) Rebuild(newTrunkName string) error {
	return nil
}

// PRManager interface implementation

func (e *DemoEngine) GetPrInfo(branchName string) (*engine.PrInfo, error) {
	if info, exists := e.prInfoMap[branchName]; exists {
		return info, nil
	}
	return nil, nil
}

func (e *DemoEngine) UpsertPrInfo(branchName string, prInfo *engine.PrInfo) error {
	e.prInfoMap[branchName] = prInfo
	return nil
}

// SyncManager interface implementation

func (e *DemoEngine) BranchMatchesRemote(branchName string) (bool, error) {
	return true, nil // Demo branches always match remote
}

func (e *DemoEngine) PopulateRemoteShas() error {
	return nil // No-op for demo
}

func (e *DemoEngine) PullTrunk() (engine.PullResult, error) {
	return engine.PullUnneeded, nil
}

func (e *DemoEngine) ResetTrunkToRemote() error {
	return nil
}

func (e *DemoEngine) RestackBranch(branchName string) (engine.RestackBranchResult, error) {
	return engine.RestackBranchResult{
		Result: engine.RestackUnneeded,
	}, nil
}

func (e *DemoEngine) ContinueRebase(rebasedBranchBase string) (engine.ContinueRebaseResult, error) {
	return engine.ContinueRebaseResult{
		Result:     0, // RebaseDone
		BranchName: GetDemoCurrentBranch(),
	}, nil
}

// SquashManager interface implementation

func (e *DemoEngine) SquashCurrentBranch(opts engine.SquashOptions) error {
	return nil
}

// SplitManager interface implementation

func (e *DemoEngine) GetAllCommits(branchName string, format engine.CommitFormat) ([]string, error) {
	return []string{
		"abc1234 Initial commit for " + branchName,
		"def5678 Add feature implementation",
	}, nil
}

func (e *DemoEngine) ApplySplitToCommits(opts engine.ApplySplitOptions) error {
	return nil
}

func (e *DemoEngine) Detach(revision string) error {
	return nil
}

func (e *DemoEngine) DetachAndResetBranchChanges(branchName string) error {
	return nil
}

func (e *DemoEngine) ForceCheckoutBranch(branchName string) error {
	return nil
}

// Ensure DemoEngine implements engine.Engine
var _ engine.Engine = (*DemoEngine)(nil)
