package engine

import (
	"context"
	"iter"
	"time"
)

// BranchReader defines the interface that Branch needs from its reader
// This is implemented by types in the engine package
type BranchReader interface {
	// State queries
	AllBranches() []Branch              // Returns all branches
	CurrentBranch() *Branch             // Returns current branch (nil if not on a branch)
	Trunk() Branch                      // Returns the trunk branch
	GetBranch(branchName string) Branch // Returns a Branch wrapper
	GetParent(branch Branch) *Branch    // Returns nil if no parent
	GetRelativeStack(branch Branch, rng StackRange) []Branch

	// Stack queries
	GetRelativeStackUpstack(branch Branch) []Branch
	GetRelativeStackDownstack(branch Branch) []Branch
	GetFullStack(branch Branch) []Branch
	SortBranchesTopologically(branches []Branch) []Branch
	IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error)
	IsBranchEmpty(ctx context.Context, branchName string) (bool, error)

	// Internal methods used by Branch type (exported so implementations outside this package can provide them)
	IsTrunkInternal(branchName string) bool
	IsBranchTrackedInternal(branchName string) bool
	IsBranchUpToDateInternal(branchName string) bool                                // Internal method for Branch type
	GetScopeInternal(branchName string) Scope                                       // Internal method for Branch type
	GetExplicitScopeInternal(branchName string) Scope                               // Internal method for Branch type
	GetChildrenInternal(branchName string) []Branch                                 // Internal method for Branch type
	GetCommitDateInternal(branchName string) (time.Time, error)                     // Internal method for Branch type
	GetCommitAuthorInternal(branchName string) (string, error)                      // Internal method for Branch type
	GetRevisionInternal(branchName string) (string, error)                          // Internal method for Branch type
	GetAllCommitsInternal(branchName string, format CommitFormat) ([]string, error) // Internal method for Branch type
	GetRelativeStackInternal(branchName string, rng StackRange) []Branch            // Internal method for Branch type

	// Commit information
	FindBranchForCommit(commitSHA string) (string, error)

	// Traversal
	BranchesDepthFirst(startBranch Branch) iter.Seq2[Branch, int]

	// Status queries
	GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error)
	FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error)
}

// BranchWriter provides write operations for branch management
type BranchWriter interface {
	// Branch tracking
	TrackBranch(ctx context.Context, branchName string, parentBranchName string) error
	UntrackBranch(branchName string) error
	SetParent(ctx context.Context, branchName string, parentBranchName string) error
	SetScope(branch Branch, scope Scope) error
	RenameBranch(ctx context.Context, oldBranch, newBranch Branch) error
	DeleteBranch(ctx context.Context, branchName string) error
	DeleteBranches(ctx context.Context, branchNames []string) ([]string, error)

	// Initialization operations
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error
}
