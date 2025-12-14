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
	parentMap     map[string]string // branch -> parent
	childrenMap   map[string][]string // branch -> children
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
// If RecursiveParents is true, returns all ancestors up to trunk
// Otherwise, returns just the immediate parent
func (e *engineImpl) GetRelativeStack(branchName string, scope Scope) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []string{}
	current := branchName

	// Walk up the parent chain
	for {
		if current == e.trunk {
			break
		}
		parent, ok := e.parentMap[current]
		if !ok {
			break
		}
		result = append([]string{parent}, result...)
		current = parent
		if !scope.RecursiveParents {
			break
		}
	}

	return result
}

// IsTrunk checks if a branch is the trunk
func (e *engineImpl) IsTrunk(branchName string) bool {
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
// For now, always return true (simplified)
func (e *engineImpl) IsBranchFixed(branchName string) bool {
	// TODO: Implement proper logic to check if branch needs restack
	return true
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
	}

	return prInfo, nil
}

// BranchMatchesRemote checks if a branch matches its remote
// For now, always return true (simplified)
func (e *engineImpl) BranchMatchesRemote(branchName string) (bool, error) {
	// TODO: Implement proper remote comparison
	return true, nil
}

// PopulateRemoteShas populates remote branch information
// For now, no-op (simplified)
func (e *engineImpl) PopulateRemoteShas() error {
	// TODO: Implement remote SHA population
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
