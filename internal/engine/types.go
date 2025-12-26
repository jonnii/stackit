package engine

import (
	"encoding/json"
	"time"
)

// StackRange specifies the range of branches to include in stack operations
type StackRange struct {
	RecursiveParents  bool
	IncludeCurrent    bool
	RecursiveChildren bool
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

// Scope represents a branch scope that can be empty, a regular scope, or an inheritance breaker
type Scope struct {
	value string
}

// NewScope creates a new scope with the given value
func NewScope(value string) Scope {
	return Scope{value: value}
}

// Empty returns an empty scope
func Empty() Scope {
	return Scope{value: ""}
}

// None returns a scope that breaks inheritance
func None() Scope {
	return Scope{value: "none"}
}

// String returns the string representation of the scope
func (s Scope) String() string {
	return s.value
}

// IsEmpty returns true if the scope is empty
func (s Scope) IsEmpty() bool {
	return s.value == ""
}

// IsNone returns true if the scope breaks inheritance
func (s Scope) IsNone() bool {
	return s.value == "none" || s.value == "clear"
}

// IsDefined returns true if the scope has a meaningful value (not empty and not none)
func (s Scope) IsDefined() bool {
	return !s.IsEmpty() && !s.IsNone()
}

// Equal checks if two scopes are equal
func (s Scope) Equal(other Scope) bool {
	return s.value == other.value
}

// MarshalJSON implements json.Marshaler
func (s Scope) MarshalJSON() ([]byte, error) {
	if s.IsEmpty() {
		return []byte("null"), nil
	}
	return json.Marshal(s.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (s *Scope) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = Empty()
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = NewScope(str)
	return nil
}

// DeletionStatus represents the deletion status of a branch
type DeletionStatus struct {
	SafeToDelete bool   // True if the branch is merged, closed, or empty (with PR)
	Reason       string // Reason why it's safe (or not) to delete
}

// Branch represents a branch in the stack
type Branch struct {
	Name   string
	Reader BranchReader
}

// GetName returns the branch name. This method allows Branch to implement
// the engine.Branch interface without creating circular dependencies.
func (b Branch) GetName() string {
	return b.Name
}

// Equal checks if two branches are equal by comparing their names
func (b Branch) Equal(other Branch) bool {
	return b.Name == other.Name
}

// IsTrunk checks if this branch is the trunk
func (b Branch) IsTrunk() bool {
	return b.Reader.IsTrunkInternal(b.Name)
}

// IsTracked checks if this branch is tracked (has metadata)
func (b Branch) IsTracked() bool {
	return b.Reader.IsBranchTrackedInternal(b.Name)
}

// GetScope returns the scope for this branch, inheriting from parent if not set
func (b Branch) GetScope() Scope {
	return b.Reader.GetScopeInternal(b.Name)
}

// GetChildren returns the children branches
func (b Branch) GetChildren() []Branch {
	return b.Reader.GetChildrenInternal(b.Name)
}

// GetParentPrecondition returns the parent branch name, or trunk if no parent
// This is used for validation where we expect a parent to exist
func (b Branch) GetParentPrecondition() string {
	parent := b.Reader.GetParent(b)
	if parent == nil {
		return b.Reader.Trunk().Name
	}
	return parent.Name
}

// IsBranchUpToDate checks if this branch is up to date with its parent
// A branch is up to date if its parent revision matches the stored parent revision
func (b Branch) IsBranchUpToDate() bool {
	return b.Reader.IsBranchUpToDateInternal(b.Name)
}

// GetRelativeStack returns the stack relative to this branch
func (b Branch) GetRelativeStack(scope StackRange) []Branch {
	return b.Reader.GetRelativeStackInternal(b.Name, scope)
}

// GetCommitDate returns the commit date for this branch
func (b Branch) GetCommitDate() (time.Time, error) {
	return b.Reader.GetCommitDateInternal(b.Name)
}

// GetCommitAuthor returns the commit author for this branch
func (b Branch) GetCommitAuthor() (string, error) {
	return b.Reader.GetCommitAuthorInternal(b.Name)
}

// GetRevision returns the SHA of this branch
func (b Branch) GetRevision() (string, error) {
	return b.Reader.GetRevisionInternal(b.Name)
}

// GetCommitCount returns the number of commits for this branch
func (b Branch) GetCommitCount() (int, error) {
	return b.Reader.GetCommitCountInternal(b.Name)
}

// GetDiffStats returns the number of lines added and deleted for this branch
func (b Branch) GetDiffStats() (added int, deleted int, err error) {
	return b.Reader.GetDiffStatsInternal(b.Name)
}

// GetAllCommits returns commits for this branch in various formats
func (b Branch) GetAllCommits(format CommitFormat) ([]string, error) {
	return b.Reader.GetAllCommitsInternal(b.Name, format)
}

// PrInfo represents PR information for a branch
type PrInfo struct {
	Number  *int
	Title   string
	Body    string
	IsDraft bool
	State   string // MERGED, CLOSED, OPEN
	Base    string // Base branch name
	URL     string // PR URL
}

// ValidationResult represents the validation state of a branch
type ValidationResult int

const (
	// ValidationResultValid indicates the branch is valid
	ValidationResultValid ValidationResult = iota
	// ValidationResultInvalidParent indicates the branch has an invalid parent
	ValidationResultInvalidParent
	// ValidationResultBadParentRevision indicates the branch has a bad parent revision
	ValidationResultBadParentRevision
	// ValidationResultBadParentName indicates the branch has a bad parent name
	ValidationResultBadParentName
	// ValidationResultTrunk indicates the branch is a trunk
	ValidationResultTrunk
)

// PullResult represents the result of pulling trunk
type PullResult int

const (
	// PullDone indicates the pull was successful
	PullDone PullResult = iota
	// PullUnneeded indicates no pull was needed
	PullUnneeded
	// PullConflict indicates a conflict occurred during pull
	PullConflict
)

// RestackResult represents the result of restacking a branch
type RestackResult int

const (
	// RestackDone indicates the restack was successful
	RestackDone RestackResult = iota
	// RestackUnneeded indicates no restack was needed
	RestackUnneeded
	// RestackConflict indicates a conflict occurred during restack
	RestackConflict
)

// RestackBranchResult represents the result of restacking a branch, including the rebased branch base
type RestackBranchResult struct {
	Result            RestackResult
	RebasedBranchBase string // The new parent revision after successful rebase (only set if Result is RestackDone or RestackConflict)
	Reparented        bool   // True if the branch was reparented due to merged/deleted parent
	OldParent         string // The old parent branch name (only set if Reparented is true)
	NewParent         string // The new parent branch name (only set if Reparented is true)
}

// RestackBatchResult represents the result of restacking multiple branches
type RestackBatchResult struct {
	ConflictBranch    string                         // The branch that hit a conflict
	RebasedBranchBase string                         // The parent revision for the conflict
	RemainingBranches []string                       // Branches that weren't reached
	Results           map[string]RestackBranchResult // Results for each branch attempted
}

// ContinueRebaseResult represents the result of continuing a rebase
type ContinueRebaseResult struct {
	Result     int    // git.RebaseResult value (0 = RebaseDone, 1 = RebaseConflict)
	BranchName string // Only set if Result is RebaseDone
}

// PRSubmissionStatus represents the submission status of a branch
type PRSubmissionStatus struct {
	Action      string // "create", "update", or "skip"
	NeedsUpdate bool   // True if the branch has changes or metadata needs update
	Reason      string // Reason for the status
	PRNumber    *int
	PRInfo      *PrInfo
}

// SquashOptions contains options for squashing commits
type SquashOptions struct {
	Message string
	NoEdit  bool
}
