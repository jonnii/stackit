// Package engine provides the core branch state management interface and implementation.
// It tracks branch relationships, metadata, and provides operations for querying
// and manipulating the branch stack.
package engine

import (
	"context"
	"time"
)

// BranchReader provides read-only access to branch information
// Thread-safe: All methods are safe for concurrent use
type BranchReader interface {
	// State queries
	AllBranchNames() []string
	CurrentBranch() string
	Trunk() string
	GetParent(branchName string) string             // Returns empty string if no parent
	GetParentPrecondition(branchName string) string // Returns parent, panics if no parent (for submit validation)
	GetChildren(branchName string) []string
	GetRelativeStack(branchName string, scope Scope) []string
	IsTrunk(branchName string) bool
	IsBranchTracked(branchName string) bool
	IsBranchFixed(ctx context.Context, branchName string) bool

	// Commit information
	GetCommitDate(ctx context.Context, branchName string) (time.Time, error)
	GetCommitAuthor(ctx context.Context, branchName string) (string, error)
	GetRevision(ctx context.Context, branchName string) (string, error)

	// Stack queries
	GetRelativeStackUpstack(branchName string) []string
	IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error)
	IsBranchEmpty(ctx context.Context, branchName string) (bool, error)
}

// BranchWriter provides write operations for branch management
// Thread-safe: All methods are safe for concurrent use
type BranchWriter interface {
	// Branch tracking
	TrackBranch(ctx context.Context, branchName string, parentBranchName string) error
	SetParent(ctx context.Context, branchName string, parentBranchName string) error
	DeleteBranch(ctx context.Context, branchName string) error

	// Initialization operations
	Reset(ctx context.Context, newTrunkName string) error
	Rebuild(ctx context.Context, newTrunkName string) error
}

// PRManager provides operations for managing pull request information
// Thread-safe: All methods are safe for concurrent use
type PRManager interface {
	GetPrInfo(ctx context.Context, branchName string) (*PrInfo, error)
	UpsertPrInfo(ctx context.Context, branchName string, prInfo *PrInfo) error
}

// SyncManager provides operations for syncing and restacking branches
// Thread-safe: All methods are safe for concurrent use
type SyncManager interface {
	// Remote operations
	BranchMatchesRemote(ctx context.Context, branchName string) (bool, error)
	PopulateRemoteShas(ctx context.Context) error
	PushBranch(ctx context.Context, branchName string, remote string, force bool, forceWithLease bool) error

	// Sync operations
	PullTrunk(ctx context.Context) (PullResult, error)
	ResetTrunkToRemote(ctx context.Context) error
	RestackBranch(ctx context.Context, branchName string) (RestackBranchResult, error)
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
	// GetAllCommits returns commits for a branch in various formats
	GetAllCommits(ctx context.Context, branchName string, format CommitFormat) ([]string, error)

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
	CommitFormatSHA      CommitFormat = "SHA"      // Full SHA
	CommitFormatReadable CommitFormat = "READABLE" // Oneline format: "abc123 Commit message"
	CommitFormatMessage  CommitFormat = "MESSAGE"  // Full commit message
	CommitFormatSubject  CommitFormat = "SUBJECT"  // First line of commit message
)

// ApplySplitOptions contains options for applying a split
type ApplySplitOptions struct {
	BranchToSplit string   // The branch being split
	BranchNames   []string // Branch names from oldest to newest
	BranchPoints  []int    // Commit indices (0 = HEAD, 1 = HEAD~1, etc.)
}

// Engine is the core interface for branch state management
// It composes BranchReader, BranchWriter, PRManager, SyncManager, SquashManager, and SplitManager
// for backward compatibility. New code should prefer using the smaller interfaces.
// Thread-safe: All methods are safe for concurrent use
type Engine interface {
	BranchReader
	BranchWriter
	PRManager
	SyncManager
	SquashManager
	SplitManager
}
