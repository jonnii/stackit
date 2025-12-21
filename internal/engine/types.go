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
