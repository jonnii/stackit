// Package engine provides the core branch state management interface and implementation.
// It tracks branch relationships, metadata, and provides operations for querying
// and manipulating the branch stack.
package engine

import (
	"context"
	"iter"
	"time"
)

// Branch represents a branch in the stack
type Branch struct {
	Name   string
	Reader BranchReader
}

// IsTrunk checks if this branch is the trunk
func (b Branch) IsTrunk() bool {
	return b.Reader.IsTrunkInternal(b.Name)
}

// IsTracked checks if this branch is tracked (has metadata)
func (b Branch) IsTracked() bool {
	return b.Reader.IsBranchTrackedInternal(b.Name)
}

// BranchReader provides read-only access to branch information
// Thread-safe: All methods are safe for concurrent use
type BranchReader interface {
	// State queries
	AllBranches() []Branch                          // Returns all branches
	CurrentBranch() Branch                          // Returns current branch (Name is empty if not on a branch)
	Trunk() Branch                                  // Returns the trunk branch
	GetBranch(branchName string) Branch             // Returns a Branch wrapper
	GetParent(branchName string) string             // Returns empty string if no parent
	GetParentPrecondition(branchName string) string // Returns parent, panics if no parent (for submit validation)
	GetChildren(branchName string) []string
	GetRelativeStack(branchName string, scope Scope) []string
	IsBranchUpToDate(branchName string) bool

	// Internal methods used by Branch type (exported so implementations outside this package can provide them)
	IsTrunkInternal(branchName string) bool
	IsBranchTrackedInternal(branchName string) bool

	// Commit information
	GetCommitDate(branchName string) (time.Time, error)
	GetCommitAuthor(branchName string) (string, error)
	GetRevision(branchName string) (string, error)
	GetAllCommits(branchName string, format CommitFormat) ([]string, error)
	FindBranchForCommit(commitSHA string) (string, error)

	// Stack queries
	GetRelativeStackUpstack(branchName string) []string
	GetRelativeStackDownstack(branchName string) []string
	GetFullStack(branchName string) []string
	SortBranchesTopologically(branches []string) []string
	IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error)
	IsBranchEmpty(ctx context.Context, branchName string) (bool, error)
	FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error)
	GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error)

	// Traversal
	BranchesDepthFirst(startBranch string) iter.Seq2[string, int]
}

// BranchWriter provides write operations for branch management
// Thread-safe: All methods are safe for concurrent use
type BranchWriter interface {
	// Branch tracking
	TrackBranch(ctx context.Context, branchName string, parentBranchName string) error
	UntrackBranch(branchName string) error
	SetParent(ctx context.Context, branchName string, parentBranchName string) error
	DeleteBranch(ctx context.Context, branchName string) error
	DeleteBranches(ctx context.Context, branchNames []string) ([]string, error)

	// Initialization operations
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error
}

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
	RestackBranch(ctx context.Context, branchName string) (RestackBranchResult, error)
	RestackBranches(ctx context.Context, branchNames []string) (RestackBatchResult, error)
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

// CommitFormat specifies the format for commit output
type CommitFormat string

const (
	// CommitFormatSHA is the full commit SHA
	CommitFormatSHA CommitFormat = "SHA" // Full SHA
	// CommitFormatReadable is a readable one-line format
	CommitFormatReadable CommitFormat = "READABLE" // Oneline format: "abc123 Commit message"
	// CommitFormatMessage is the full commit message
	CommitFormatMessage CommitFormat = "MESSAGE" // Full commit message
	// CommitFormatSubject is the first line of the commit message
	CommitFormatSubject CommitFormat = "SUBJECT" // First line of commit message
)

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
	TakeSnapshot(command string, args []string) error
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
