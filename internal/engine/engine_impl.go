package engine

import (
	"fmt"
	"sync"

	"stackit.dev/stackit/internal/git"
)

// engineImpl is a minimal implementation of the Engine interface
type engineImpl struct {
	repoRoot          string
	trunk             string
	currentBranch     string
	branches          []string
	parentMap         map[string]string   // branch -> parent
	childrenMap       map[string][]string // branch -> children
	remoteShas        map[string]string   // branch -> remote SHA (populated by PopulateRemoteShas)
	maxUndoStackDepth int
	mu                sync.RWMutex
}

// NewEngine creates a new engine instance
func NewEngine(opts Options) (Engine, error) {
	// Initialize git repository
	if err := git.InitDefaultRepo(); err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Validate repo root
	if opts.RepoRoot == "" {
		return nil, fmt.Errorf("repo root must be specified in Options")
	}

	// Validate trunk
	if opts.Trunk == "" {
		return nil, fmt.Errorf("trunk must be specified in Options")
	}

	// Validate and set max undo stack depth
	maxDepth := opts.MaxUndoStackDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxUndoStackDepth
	}

	e := &engineImpl{
		repoRoot:          opts.RepoRoot,
		trunk:             opts.Trunk,
		parentMap:         make(map[string]string),
		childrenMap:       make(map[string][]string),
		remoteShas:        make(map[string]string),
		maxUndoStackDepth: maxDepth,
	}

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
	// Refresh currentBranch since we're resetting everything
	return e.rebuildInternal(true)
}

// Rebuild reloads branch cache with new trunk
func (e *engineImpl) Rebuild(newTrunkName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update trunk
	e.trunk = newTrunkName

	// Rebuild cache (already holding lock, so call rebuildInternal)
	// Refresh currentBranch since we might have switched branches
	return e.rebuildInternal(true)
}

// PopulateRemoteShas populates remote branch information by fetching SHAs from remote
func (e *engineImpl) PopulateRemoteShas() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear existing remote SHAs
	e.remoteShas = make(map[string]string)

	// Fetch remote SHAs using git ls-remote
	remote := git.GetRemote()
	remoteShas, err := git.FetchRemoteShas(remote)
	if err != nil {
		// Don't fail if we can't fetch remote SHAs (e.g., offline)
		return nil
	}

	e.remoteShas = remoteShas
	return nil
}
