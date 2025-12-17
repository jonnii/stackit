package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// ContinueOptions contains options for the continue command
type ContinueOptions struct {
	AddAll bool
}

// ContinueAction performs the continue operation
func ContinueAction(ctx *runtime.Context, opts ContinueOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Check if rebase is in progress
	if !git.IsRebaseInProgress() {
		// Clear any stale continuation state
		_ = config.ClearContinuationState(ctx.RepoRoot)
		return fmt.Errorf("no rebase in progress. Nothing to continue")
	}

	// Load continuation state
	continuation, err := config.GetContinuationState(ctx.RepoRoot)
	if err != nil {
		// No continuation state - this is okay, we can still continue the rebase
		// but we won't be able to resume restacking
		splog.Info("No continuation state found. Continuing rebase only.")
		// Try to continue the rebase anyway (user might have started it manually)
		// But we need a rebasedBranchBase - try to get it from current branch's parent
		currentBranch := eng.CurrentBranch()
		if currentBranch == "" {
			return fmt.Errorf("not on a branch")
		}
		parent := eng.GetParent(currentBranch)
		if parent == "" {
			parent = eng.Trunk()
		}
		parentRev, err := eng.GetRevision(parent)
		if err != nil {
			return fmt.Errorf("failed to get parent revision: %w", err)
		}
		continuation = &config.ContinuationState{
			RebasedBranchBase:     parentRev,
			BranchesToRestack:     []string{},
			CurrentBranchOverride: currentBranch,
		}
	}

	// Stage all changes if --all flag is set
	if opts.AddAll {
		if err := git.AddAll(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
	}

	// Continue the rebase
	result, err := eng.ContinueRebase(continuation.RebasedBranchBase)
	if err != nil {
		return fmt.Errorf("failed to continue rebase: %w", err)
	}

	// Handle result
	if result.Result == int(git.RebaseConflict) {
		// Another conflict - persist state again
		if err := config.PersistContinuationState(ctx.RepoRoot, continuation); err != nil {
			return fmt.Errorf("failed to persist continuation: %w", err)
		}
		// Get current branch name for conflict status
		branchName := result.BranchName
		if branchName == "" {
			branchName = eng.CurrentBranch()
		}
		if err := PrintConflictStatus(branchName, eng, splog); err != nil {
			return fmt.Errorf("failed to print conflict status: %w", err)
		}
		return fmt.Errorf("rebase conflict is not yet resolved")
	}

	// Success - inform user
	splog.Info("Resolved rebase conflict for %s.", tui.ColorBranchName(result.BranchName, true))

	// Continue with remaining branches to restack
	if len(continuation.BranchesToRestack) > 0 {
		if err := RestackBranches(continuation.BranchesToRestack, eng, splog, ctx.RepoRoot); err != nil {
			return err
		}
	}

	// Clear continuation state
	if err := config.ClearContinuationState(ctx.RepoRoot); err != nil {
		splog.Debug("Failed to clear continuation state: %v", err)
	}

	return nil
}
