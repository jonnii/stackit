package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// ContinueOptions are options for the continue command
type ContinueOptions struct {
	AddAll   bool
	Engine   engine.Engine
	Splog    *output.Splog
	RepoRoot string
}

// ContinueAction performs the continue operation
func ContinueAction(opts ContinueOptions) error {
	// Check if rebase is in progress
	if !git.IsRebaseInProgress() {
		// Clear any stale continuation state
		_ = config.ClearContinuationState(opts.RepoRoot)
		return fmt.Errorf("no rebase in progress. Nothing to continue")
	}

	// Load continuation state
	continuation, err := config.GetContinuationState(opts.RepoRoot)
	if err != nil {
		// No continuation state - just continue the rebase
		return fmt.Errorf("no continuation state found. Use 'git rebase --continue' directly")
	}

	// Stage all changes if --all flag is set
	if opts.AddAll {
		if err := git.AddAll(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
	}

	// Continue the rebase
	result, err := opts.Engine.ContinueRebase(continuation.RebasedBranchBase)
	if err != nil {
		return fmt.Errorf("failed to continue rebase: %w", err)
	}

	// Handle result
	if result.Result == int(git.RebaseConflict) {
		// Another conflict - persist state again
		if err := config.PersistContinuationState(opts.RepoRoot, continuation); err != nil {
			return fmt.Errorf("failed to persist continuation: %w", err)
		}
		// Get current branch name for conflict status
		branchName := result.BranchName
		if branchName == "" {
			branchName = opts.Engine.CurrentBranch()
		}
		if err := PrintConflictStatus(branchName, opts.Engine, opts.Splog); err != nil {
			return fmt.Errorf("failed to print conflict status: %w", err)
		}
		return fmt.Errorf("rebase conflict is not yet resolved")
	}

	// Success - inform user
	opts.Splog.Info("Resolved rebase conflict for %s.", output.ColorBranchName(result.BranchName, true))

	// Continue with remaining branches to restack
	if len(continuation.BranchesToRestack) > 0 {
		if err := RestackBranches(continuation.BranchesToRestack, opts.Engine, opts.Splog, opts.RepoRoot); err != nil {
			return err
		}
	}

	// Clear continuation state
	if err := config.ClearContinuationState(opts.RepoRoot); err != nil {
		opts.Splog.Debug("Failed to clear continuation state: %v", err)
	}

	return nil
}
