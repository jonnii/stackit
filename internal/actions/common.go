package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
)

// Restacker is a minimal interface needed for restacking branches
type Restacker interface {
	engine.BranchReader
	engine.SyncManager
}

// RestackBranches restacks a list of branches using the engine's batch restack method
func RestackBranches(ctx context.Context, branches []engine.Branch, eng Restacker, splog *tui.Splog) error {
	// Use the engine's optimized batch restack method
	batchResult, err := eng.RestackBranches(ctx, branches)
	if err != nil {
		return fmt.Errorf("batch restack failed: %w", err)
	}

	// Process results and provide user feedback
	currentBranch := eng.CurrentBranch()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.Name
	}

	for _, branch := range branches {
		branchName := branch.Name
		result, exists := batchResult.Results[branchName]
		if !exists {
			continue // Skip branches not processed (e.g., trunk)
		}

		// Log reparenting if it happened (batch version doesn't track this, so we skip for now)
		// TODO: Consider adding reparenting info to RestackBatchResult if needed

		switch result.Result {
		case engine.RestackDone:
			parent := eng.GetParent(branch)
			parentName := ""
			if parent == nil {
				parentName = eng.Trunk().Name
			} else {
				parentName = parent.Name
			}
			isCurrent := branchName == currentBranchName
			splog.Info("Restacked %s on %s.",
				tui.ColorBranchName(branchName, isCurrent),
				tui.ColorBranchName(parentName, false))
		case engine.RestackConflict:
			// This should have been handled by the batch method returning an error
			// If we get here, it means there was a conflict but batch processing continued
			return fmt.Errorf("unexpected conflict state for branch %s", branchName)
		case engine.RestackUnneeded:
			if branch.IsTrunk() {
				splog.Info("%s does not need to be restacked.", tui.ColorBranchName(branchName, false))
			} else {
				parent := eng.GetParent(branch)
				parentName := ""
				if parent == nil {
					parentName = eng.Trunk().Name
				} else {
					parentName = parent.Name
				}
				isCurrent := branchName == currentBranchName
				splog.Info("%s does not need to be restacked on %s.",
					tui.ColorBranchName(branchName, isCurrent),
					tui.ColorBranchName(parentName, false))
			}
		}
	}

	return nil
}

// PluralSuffix returns "es" if plural is true, otherwise empty string
func PluralSuffix(plural bool) string {
	if plural {
		return "es"
	}
	return ""
}

// Pluralize returns the word with "ren" suffix if count != 1 (specific to "child" -> "children")
func Pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "ren" // "child" -> "children"
}

// ShouldDeleteBranch checks if a branch should be deleted
func ShouldDeleteBranch(ctx context.Context, branchName string, eng engine.Engine, force bool) (bool, string) {
	status, err := eng.GetDeletionStatus(ctx, branchName)
	if err != nil {
		return false, ""
	}

	if status.SafeToDelete {
		return true, status.Reason
	}

	// If force, don't prompt
	if force {
		return false, ""
	}

	// For now, we don't prompt interactively
	// In a full implementation, we would prompt here
	return false, ""
}

// PluralIt returns "them" if plural is true, otherwise "it"
func PluralIt(plural bool) string {
	if plural {
		return "them"
	}
	return "it"
}

// SnapshotOption is a function that modifies SnapshotOptions
type SnapshotOption func(*engine.SnapshotOptions)

// NewSnapshot creates a new SnapshotOptions with the given command and options
func NewSnapshot(command string, options ...SnapshotOption) engine.SnapshotOptions {
	opts := engine.SnapshotOptions{
		Command: command,
		Args:    []string{},
	}
	for _, option := range options {
		option(&opts)
	}
	return opts
}

// WithArg appends a single argument if it's not empty
func WithArg(arg string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if arg != "" {
			opts.Args = append(opts.Args, arg)
		}
	}
}

// WithArgs appends multiple arguments
func WithArgs(args ...string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		opts.Args = append(opts.Args, args...)
	}
}

// WithFlag appends a flag if condition is true
func WithFlag(condition bool, flag string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if condition {
			opts.Args = append(opts.Args, flag)
		}
	}
}

// WithFlagValue appends a flag with a value if the value is not empty
func WithFlagValue(flag string, value string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if value != "" {
			opts.Args = append(opts.Args, flag, value)
		}
	}
}
