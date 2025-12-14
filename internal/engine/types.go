package engine

// PrInfo represents PR information for a branch
type PrInfo struct {
	Number  *int
	Title   string
	Body    string
	IsDraft bool
}

// Scope specifies the scope for stack operations
type Scope struct {
	RecursiveParents bool
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
