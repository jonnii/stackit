package engine

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

// Scope specifies the scope for stack operations
type Scope struct {
	RecursiveParents  bool
	IncludeCurrent    bool
	RecursiveChildren bool
}

// ValidationResult represents the validation state of a branch
type ValidationResult int

const (
	ValidationResultValid ValidationResult = iota
	ValidationResultInvalidParent
	ValidationResultBadParentRevision
	ValidationResultBadParentName
	ValidationResultTrunk
)

// PullResult represents the result of pulling trunk
type PullResult int

const (
	PullDone PullResult = iota
	PullUnneeded
	PullConflict
)

// RestackResult represents the result of restacking a branch
type RestackResult int

const (
	RestackDone RestackResult = iota
	RestackUnneeded
	RestackConflict
)

// RestackBranchResult represents the result of restacking a branch, including the rebased branch base
type RestackBranchResult struct {
	Result            RestackResult
	RebasedBranchBase string // The new parent revision after successful rebase (only set if Result is RestackDone or RestackConflict)
}

// ContinueRebaseResult represents the result of continuing a rebase
type ContinueRebaseResult struct {
	Result     int    // git.RebaseResult value (0 = RebaseDone, 1 = RebaseConflict)
	BranchName string // Only set if Result is RebaseDone
}

// SquashOptions contains options for squashing commits
type SquashOptions struct {
	Message string
	NoEdit  bool
}
