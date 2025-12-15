package engine

import (
	"time"
)

// Engine is the core interface for branch state management
// This interface defines the methods needed for the log command
type Engine interface {
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
	IsBranchFixed(branchName string) bool

	// Commit information
	GetCommitDate(branchName string) (time.Time, error)
	GetCommitAuthor(branchName string) (string, error)
	GetRevision(branchName string) (string, error)

	// PR information
	GetPrInfo(branchName string) (*PrInfo, error)
	UpsertPrInfo(branchName string, prInfo *PrInfo) error

	// Remote operations
	BranchMatchesRemote(branchName string) (bool, error)
	PopulateRemoteShas() error

	// Initialization operations
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error

	// Branch tracking
	TrackBranch(branchName string, parentBranchName string) error

	// Sync operations
	PullTrunk() (PullResult, error)
	ResetTrunkToRemote() error
	RestackBranch(branchName string) (RestackBranchResult, error)
	ContinueRebase(rebasedBranchBase string) (ContinueRebaseResult, error)
	IsMergedIntoTrunk(branchName string) (bool, error)
	IsBranchEmpty(branchName string) (bool, error)
	DeleteBranch(branchName string) error
	SetParent(branchName string, parentBranchName string) error
	GetRelativeStackUpstack(branchName string) []string
}
