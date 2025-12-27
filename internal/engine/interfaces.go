package engine

import (
	"context"
	"iter"
	"time"

	"stackit.dev/stackit/internal/git"
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
	GetCommitCountInternal(branchName string) (int, error)                          // Internal method for Branch type
	GetDiffStatsInternal(branchName string) (added int, deleted int, err error)     // Internal method for Branch type
	GetAllCommitsInternal(branchName string, format CommitFormat) ([]string, error) // Internal method for Branch type
	GetRelativeStackInternal(branchName string, rng StackRange) []Branch            // Internal method for Branch type

	// Commit information
	FindBranchForCommit(commitSHA string) (string, error)

	// Traversal
	BranchesDepthFirst(startBranch Branch) iter.Seq2[Branch, int]

	// Status queries
	GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error)
	FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error)
	ListMetadataRefs() (map[string]string, error)
	BatchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error)
	ReadMetadataRef(branchName string) (*Meta, error)
	GetRemote() string
	GetBranchRemoteDifference(branchName string) (string, error)

	// Low-level Git state queries
	HasStagedChanges(ctx context.Context) (bool, error)
	HasUnstagedChanges(ctx context.Context) (bool, error)
	GetMergeBase(rev1, rev2 string) (string, error)
	GetChangedFiles(ctx context.Context, base, head string) ([]string, error)
	ParseStagedHunks(ctx context.Context) ([]git.Hunk, error)
	ShowDiff(ctx context.Context, left, right string, stat bool) (string, error)
	ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error)
	GetUnmergedFiles(ctx context.Context) ([]string, error)
	GetParentCommitSHA(commitSHA string) (string, error)
	GetCommitSHA(branchName string, offset int) (string, error)
	IsAncestor(ancestor, descendant string) (bool, error)

	// Worktree operations
	ListWorktrees(ctx context.Context) ([]string, error)
	GetWorkingDir() string

	// Git read operations
	RunGitCommandWithContext(ctx context.Context, args ...string) (string, error)
	RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error)
}

// BranchWriter provides write operations for branch management
type BranchWriter interface {
	// Branch tracking
	TrackBranch(ctx context.Context, branchName string, parentBranchName string) error
	UntrackBranch(branchName string) error
	SetParent(ctx context.Context, branch Branch, parentBranch Branch) error
	UpdateParentRevision(branchName string, parentRev string) error
	SetScope(branch Branch, scope Scope) error
	RenameBranch(ctx context.Context, oldBranch, newBranch Branch) error
	DeleteBranch(ctx context.Context, branch Branch) error
	DeleteBranches(ctx context.Context, branches []Branch) ([]string, error)
	DeleteMetadataRef(branch Branch) error
	RenameMetadataRef(oldBranch, newBranch Branch) error
	WriteMetadataRef(branch Branch, meta *Meta) error

	// Checkout operations
	CheckoutBranch(ctx context.Context, branch Branch) error
	CreateAndCheckoutBranch(ctx context.Context, branch Branch) error

	// Git write operations
	Commit(ctx context.Context, message string, verbose int) error
	StageAll(ctx context.Context) error
	StashPush(ctx context.Context, message string) (string, error)
	StashPop(ctx context.Context) error

	// Worktree operations
	AddWorktree(ctx context.Context, path string, branch string, detach bool) error
	RemoveWorktree(ctx context.Context, path string) error
	SetWorkingDir(dir string)

	// Git low-level write operations
	RunGitCommand(args ...string) (string, error)
	RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error)

	// Initialization operations
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error
}

// AbsorbManager defines the interface for the absorb operation.
type AbsorbManager interface {
	ApplyHunksToBranch(ctx context.Context, branch Branch, hunksByCommit map[string][]git.Hunk) error
	FindTargetCommitForHunk(hunk git.Hunk, commitSHAs []string) (string, int, error)
}
