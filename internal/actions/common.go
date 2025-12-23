package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
)

// Restacker is a minimal interface needed for restacking branches
type Restacker interface {
	engine.BranchReader
	engine.SyncManager
}

// RestackBranches restacks a list of branches
func RestackBranches(ctx context.Context, branches []engine.Branch, eng Restacker, splog *tui.Splog, repoRoot string) error {
	for i, branch := range branches {
		if branch.IsTrunk() {
			splog.Info("%s does not need to be restacked.", tui.ColorBranchName(branch.Name, false))
			continue
		}

		result, err := eng.RestackBranch(ctx, branch)
		if err != nil {
			return fmt.Errorf("failed to restack %s: %w", branch.Name, err)
		}

		// Log reparenting if it happened
		if result.Reparented {
			currentBranch := eng.CurrentBranch()
			isCurrent := currentBranch != nil && branch.Name == currentBranch.Name
			splog.Info("Reparented %s from %s to %s (parent was merged/deleted).",
				tui.ColorBranchName(branch.Name, isCurrent),
				tui.ColorBranchName(result.OldParent, false),
				tui.ColorBranchName(result.NewParent, false))
		}

		switch result.Result {
		case engine.RestackDone:
			parent := eng.GetParent(branch)
			parentName := ""
			if parent == nil {
				parentName = eng.Trunk().Name
			} else {
				parentName = parent.Name
			}
			currentBranch := eng.CurrentBranch()
			isCurrent := currentBranch != nil && branch.Name == currentBranch.Name
			splog.Info("Restacked %s on %s.",
				tui.ColorBranchName(branch.Name, isCurrent),
				tui.ColorBranchName(parentName, false))
		case engine.RestackConflict:
			// Persist continuation state with remaining branches
			currentBranch := eng.CurrentBranch()
			currentBranchName := ""
			if currentBranch != nil {
				currentBranchName = currentBranch.Name
			}
			// Convert remaining branches to []string for continuation state
			remainingBranchNames := make([]string, len(branches[i+1:]))
			for j, b := range branches[i+1:] {
				remainingBranchNames[j] = b.Name
			}
			continuation := &config.ContinuationState{
				BranchesToRestack:     remainingBranchNames, // Remaining branches
				RebasedBranchBase:     result.RebasedBranchBase,
				CurrentBranchOverride: currentBranchName,
			}

			if err := config.PersistContinuationState(repoRoot, continuation); err != nil {
				return fmt.Errorf("failed to persist continuation: %w", err)
			}

			// Print conflict status
			if err := PrintConflictStatus(ctx, branch.Name, splog); err != nil {
				return fmt.Errorf("failed to print conflict status: %w", err)
			}

			return fmt.Errorf("hit conflict restacking %s", branch.Name)
		case engine.RestackUnneeded:
			parent := eng.GetParent(branch)
			parentName := ""
			if parent == nil {
				parentName = eng.Trunk().Name
			} else {
				parentName = parent.Name
			}
			currentBranch := eng.CurrentBranch()
			isCurrent := currentBranch != nil && branch.Name == currentBranch.Name
			splog.Info("%s does not need to be restacked on %s.",
				tui.ColorBranchName(branch.Name, isCurrent),
				tui.ColorBranchName(parentName, false))
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
