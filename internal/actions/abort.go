package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// AbortOptions contains options for the abort command
type AbortOptions struct {
	Force bool
}

// AbortAction cancels an in-progress operation
func AbortAction(ctx *runtime.Context, opts AbortOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	rebaseInProgress := git.IsRebaseInProgress(ctx.Context)
	mergeInProgress := git.IsMergeInProgress(ctx.Context)

	// Check for continuation state
	hasContinuation := false
	if _, err := config.GetContinuationState(ctx.RepoRoot); err == nil {
		hasContinuation = true
	}

	if !rebaseInProgress && !mergeInProgress && !hasContinuation {
		splog.Info("No operation in progress to abort.")
		return nil
	}

	// Confirm unless force is used
	if !opts.Force {
		msg := "Are you sure you want to abort the current operation? This will roll back the repository to its previous state."
		confirmed, err := tui.PromptConfirm(msg, false)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			splog.Info("Abort canceled.")
			return nil
		}
	}

	// Abort Git operations
	if rebaseInProgress {
		splog.Info("Aborting rebase...")
		if err := git.RebaseAbort(ctx.Context); err != nil {
			return fmt.Errorf("failed to abort rebase: %w", err)
		}
	}
	if mergeInProgress {
		splog.Info("Aborting merge...")
		if err := git.MergeAbort(ctx.Context); err != nil {
			return fmt.Errorf("failed to abort merge: %w", err)
		}
	}

	// Clear continuation state
	if hasContinuation {
		if err := config.ClearContinuationState(ctx.RepoRoot); err != nil {
			splog.Debug("Failed to clear continuation state: %v", err)
		}
	}

	// Restore latest snapshot
	snapshots, err := eng.GetSnapshots()
	if err != nil {
		return fmt.Errorf("failed to get snapshots: %w", err)
	}

	if len(snapshots) > 0 {
		splog.Info("Restoring to state before the command started...")
		// The latest snapshot should be the one taken before the command that halted
		if err := eng.RestoreSnapshot(ctx.Context, snapshots[0].ID); err != nil {
			return fmt.Errorf("failed to restore snapshot: %w", err)
		}
		splog.Info("Successfully aborted and restored repository state.")
	} else {
		splog.Info("Operation aborted. No undo history found to restore state.")
	}

	return nil
}
