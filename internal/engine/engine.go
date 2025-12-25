// Package engine provides the core branch state management interface and implementation.
// It tracks branch relationships, metadata, and provides operations for querying
// and manipulating the branch stack.
package engine

import (
	"context"
)

// PRManager provides operations for managing pull request information
// Thread-safe: All methods are safe for concurrent use
type PRManager interface {
	GetPrInfo(branchName string) (*PrInfo, error)
	UpsertPrInfo(branchName string, prInfo *PrInfo) error
	GetPRSubmissionStatus(branchName string) (PRSubmissionStatus, error)
}

// SyncManager provides operations for syncing and restacking branches
// Thread-safe: All methods are safe for concurrent use
type SyncManager interface {
	// Remote operations
	BranchMatchesRemote(branchName string) (bool, error)
	PopulateRemoteShas() error
	PushBranch(ctx context.Context, branchName string, remote string, force bool, forceWithLease bool) error

	// Sync operations
	PullTrunk(ctx context.Context) (PullResult, error)
	ResetTrunkToRemote(ctx context.Context) error
	RestackBranches(ctx context.Context, branches []Branch) (RestackBatchResult, error)
	ContinueRebase(ctx context.Context, rebasedBranchBase string) (ContinueRebaseResult, error)
}

// SquashManager provides operations for squashing commits
// Thread-safe: All methods are safe for concurrent use
type SquashManager interface {
	SquashCurrentBranch(ctx context.Context, opts SquashOptions) error
}

// SplitManager provides operations for splitting branches
// Thread-safe: All methods are safe for concurrent use
type SplitManager interface {
	// ApplySplitToCommits creates branches at specified commit points
	ApplySplitToCommits(ctx context.Context, opts ApplySplitOptions) error

	// Detach detaches HEAD to a specific revision
	Detach(ctx context.Context, revision string) error

	// DetachAndResetBranchChanges detaches and resets branch changes
	DetachAndResetBranchChanges(ctx context.Context, branchName string) error

	// ForceCheckoutBranch force checks out a branch
	ForceCheckoutBranch(ctx context.Context, branchName string) error
}

// ApplySplitOptions contains options for applying a split
type ApplySplitOptions struct {
	BranchToSplit string   // The branch being split
	BranchNames   []string // Branch names from oldest to newest
	BranchPoints  []int    // Commit indices (0 = HEAD, 1 = HEAD~1, etc.)
}

// Options contains configuration options for creating an Engine
type Options struct {
	// RepoRoot is the root directory of the Git repository
	RepoRoot string

	// Trunk is the primary trunk branch name (e.g., "main", "master")
	Trunk string

	// MaxUndoStackDepth is the maximum number of undo snapshots to keep.
	// If zero or negative, defaults to DefaultMaxUndoStackDepth (10).
	MaxUndoStackDepth int
}

// UndoManager provides operations for undo/redo functionality
// Thread-safe: All methods are safe for concurrent use
type UndoManager interface {
	TakeSnapshot(opts SnapshotOptions) error
	GetSnapshots() ([]SnapshotInfo, error)
	LoadSnapshot(snapshotID string) (*Snapshot, error)
	RestoreSnapshot(ctx context.Context, snapshotID string) error
}

// Engine is the core interface for branch state management
// It composes BranchReader, BranchWriter, PRManager, SyncManager, SquashManager, SplitManager, and UndoManager
// for backward compatibility. New code should prefer using the smaller interfaces.
// Thread-safe: All methods are safe for concurrent use
type Engine interface {
	BranchReader
	BranchWriter
	PRManager
	SyncManager
	SquashManager
	SplitManager
	UndoManager
}
