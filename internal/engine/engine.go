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
	GetParent(branchName string) string // Returns empty string if no parent
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

	// Remote operations
	BranchMatchesRemote(branchName string) (bool, error)
	PopulateRemoteShas() error

	// Initialization operations
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error
}
