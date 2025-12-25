package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
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
	if !git.IsRebaseInProgress(ctx.Context) {
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
		if currentBranch == nil {
			return fmt.Errorf("not on a branch")
		}
		parent := eng.GetParent(*currentBranch)
		parentName := ""
		if parent == nil {
			parentName = eng.Trunk().Name
		} else {
			parentName = parent.Name
		}
		parentBranch := eng.GetBranch(parentName)
		parentRev, err := parentBranch.GetRevision()
		if err != nil {
			return fmt.Errorf("failed to get parent revision: %w", err)
		}
		continuation = &config.ContinuationState{
			RebasedBranchBase:     parentRev,
			BranchesToRestack:     []string{},
			CurrentBranchOverride: currentBranch.Name,
		}
	}

	// Stage all changes if --all flag is set
	if opts.AddAll {
		if err := git.AddAll(ctx.Context); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
	}

	// Continue the rebase
	result, err := eng.ContinueRebase(ctx.Context, continuation.RebasedBranchBase)
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
			currentBranch := eng.CurrentBranch()
			if currentBranch == nil {
				return fmt.Errorf("not on a branch")
			}
			branchName = currentBranch.Name
		}
		if err := PrintConflictStatus(ctx.Context, branchName, splog); err != nil {
			return fmt.Errorf("failed to print conflict status: %w", err)
		}
		return fmt.Errorf("rebase conflict is not yet resolved")
	}

	// Success - inform user
	splog.Info("Resolved rebase conflict for %s.", tui.ColorBranchName(result.BranchName, true))

	// Continue with remaining branches to restack
	if len(continuation.BranchesToRestack) > 0 {
		// Convert []string to []Branch for RestackBranches
		branches := make([]engine.Branch, len(continuation.BranchesToRestack))
		for i, name := range continuation.BranchesToRestack {
			branches[i] = eng.GetBranch(name)
		}
		if err := RestackBranches(ctx.Context, branches, eng, splog); err != nil {
			return err
		}
	}

	// Clear continuation state
	if err := config.ClearContinuationState(ctx.RepoRoot); err != nil {
		splog.Debug("Failed to clear continuation state: %v", err)
	}

	return nil
}
