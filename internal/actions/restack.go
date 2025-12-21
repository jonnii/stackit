package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// RestackBranches restacks a list of branches
func RestackBranches(ctx context.Context, branchNames []string, eng engine.Engine, splog *tui.Splog, repoRoot string) error {
	for i, branchName := range branchNames {
		if eng.IsTrunk(branchName) {
			splog.Info("%s does not need to be restacked.", tui.ColorBranchName(branchName, false))
			continue
		}

		result, err := eng.RestackBranch(ctx, branchName)
		if err != nil {
			return fmt.Errorf("failed to restack %s: %w", branchName, err)
		}

		// Log reparenting if it happened
		if result.Reparented {
			splog.Info("Reparented %s from %s to %s (parent was merged/deleted).",
				tui.ColorBranchName(branchName, branchName == eng.CurrentBranch()),
				tui.ColorBranchName(result.OldParent, false),
				tui.ColorBranchName(result.NewParent, false))
		}

		switch result.Result {
		case engine.RestackDone:
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}
			splog.Info("Restacked %s on %s.",
				tui.ColorBranchName(branchName, branchName == eng.CurrentBranch()),
				tui.ColorBranchName(parent, false))
		case engine.RestackConflict:
			// Persist continuation state with remaining branches
			continuation := &config.ContinuationState{
				BranchesToRestack:     branchNames[i+1:], // Remaining branches
				RebasedBranchBase:     result.RebasedBranchBase,
				CurrentBranchOverride: eng.CurrentBranch(),
			}

			if err := config.PersistContinuationState(repoRoot, continuation); err != nil {
				return fmt.Errorf("failed to persist continuation: %w", err)
			}

			// Print conflict status
			if err := PrintConflictStatus(ctx, branchName, eng, splog); err != nil {
				return fmt.Errorf("failed to print conflict status: %w", err)
			}

			return fmt.Errorf("hit conflict restacking %s", branchName)
		case engine.RestackUnneeded:
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}
			splog.Info("%s does not need to be restacked on %s.",
				tui.ColorBranchName(branchName, branchName == eng.CurrentBranch()),
				tui.ColorBranchName(parent, false))
		}
	}

	return nil
}

// RestackOptions contains options for the restack command
type RestackOptions struct {
	BranchName string
	Scope      engine.Scope
}

// RestackAction performs the restack operation
func RestackAction(ctx *runtime.Context, opts RestackOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get branches to restack based on scope
	branches := eng.GetRelativeStack(opts.BranchName, opts.Scope)

	if len(branches) == 0 {
		splog.Info("No branches to restack.")
		return nil
	}

	// Call RestackBranches
	return RestackBranches(ctx.Context, branches, eng, splog, ctx.RepoRoot)
}
