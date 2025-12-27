package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// Restacker is a minimal interface needed for restacking branches
type Restacker interface {
	engine.BranchReader
	engine.SyncManager
}

// RestackBranches restacks a list of branches using the engine's batch restack method
func RestackBranches(ctx context.Context, branches []engine.Branch, eng Restacker, splog *tui.Splog, repoRoot string) error {
	batchResult, err := eng.RestackBranches(ctx, branches)
	if err != nil {
		if batchResult.ConflictBranch != "" {
			continuation := &config.ContinuationState{
				BranchesToRestack:     batchResult.RemainingBranches,
				RebasedBranchBase:     batchResult.RebasedBranchBase,
				CurrentBranchOverride: batchResult.ConflictBranch,
			}

			if err := config.PersistContinuationState(repoRoot, continuation); err != nil {
				return fmt.Errorf("failed to persist continuation: %w", err)
			}

			if err := PrintConflictStatus(ctx, batchResult.ConflictBranch, splog); err != nil {
				return fmt.Errorf("failed to print conflict status: %w", err)
			}
		}
		return fmt.Errorf("batch restack failed: %w", err)
	}

	// Handle conflicts even when no error was returned
	if batchResult.ConflictBranch != "" {
		continuation := &config.ContinuationState{
			BranchesToRestack:     batchResult.RemainingBranches,
			RebasedBranchBase:     batchResult.RebasedBranchBase,
			CurrentBranchOverride: batchResult.ConflictBranch,
		}

		if err := config.PersistContinuationState(repoRoot, continuation); err != nil {
			return fmt.Errorf("failed to persist continuation: %w", err)
		}

		if err := PrintConflictStatus(ctx, batchResult.ConflictBranch, splog); err != nil {
			return fmt.Errorf("failed to print conflict status: %w", err)
		}

		return fmt.Errorf("restack stopped due to conflict on %s", batchResult.ConflictBranch)
	}

	currentBranch := eng.CurrentBranch()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.GetName()
	}

	for _, branch := range branches {
		branchName := branch.GetName()
		result, exists := batchResult.Results[branchName]
		if !exists {
			continue // Skip branches not processed (e.g., trunk)
		}

		if result.Reparented {
			isCurrent := branchName == currentBranchName
			splog.Info("Reparented %s from %s to %s (parent was merged/deleted).",
				style.ColorBranchName(branchName, isCurrent),
				style.ColorBranchName(result.OldParent, false),
				style.ColorBranchName(result.NewParent, false))
		}

		switch result.Result {
		case engine.RestackDone:
			parent := eng.GetParent(branch)
			parentName := ""
			if parent == nil {
				parentName = eng.Trunk().GetName()
			} else {
				parentName = parent.GetName()
			}
			isCurrent := branchName == currentBranchName
			splog.Info("Restacked %s on %s.",
				style.ColorBranchName(branchName, isCurrent),
				style.ColorBranchName(parentName, false))
		case engine.RestackConflict:
			// This should not happen since conflicts are handled at the batch level
			return fmt.Errorf("unexpected conflict in batch result for branch %s", branchName)
		case engine.RestackUnneeded:
			if branch.IsTrunk() {
				splog.Info("%s does not need to be restacked.", style.ColorBranchName(branchName, false))
			} else {
				parent := eng.GetParent(branch)
				parentName := ""
				if parent == nil {
					parentName = eng.Trunk().GetName()
				} else {
					parentName = parent.GetName()
				}
				isCurrent := branchName == currentBranchName
				splog.Info("%s does not need to be restacked on %s.",
					style.ColorBranchName(branchName, isCurrent),
					style.ColorBranchName(parentName, false))
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

	if force {
		return false, ""
	}

	// Interactive prompting not yet implemented
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
