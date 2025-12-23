package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// ModifyOptions contains options for the modify command
type ModifyOptions struct {
	// Staging options
	All    bool // Stage all changes before committing (-a)
	Update bool // Stage updates to tracked files only (-u)
	Patch  bool // Pick hunks to stage interactively (-p)

	// Commit options
	CreateCommit bool   // Create a new commit instead of amending (-c)
	Message      string // Commit message (-m)
	Edit         bool   // Open editor to edit commit message (-e)
	NoEdit       bool   // Don't edit commit message (computed from flags)
	ResetAuthor  bool   // Reset author to current user
	Verbose      int    // Show diff in commit message template (-v)

	// Interactive rebase
	InteractiveRebase bool // Start interactive rebase on branch commits
}

// ModifyAction performs the modify operation
func ModifyAction(ctx *runtime.Context, opts ModifyOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	currentBranch, err := utils.ValidateOnBranch(ctx)
	if err != nil {
		return err
	}

	// Check if we're on trunk
	currentBranchObj := eng.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return fmt.Errorf("cannot modify trunk branch %s", currentBranch)
	}

	// Handle interactive rebase separately
	if opts.InteractiveRebase {
		return interactiveRebaseAction(ctx, opts)
	}

	// Check if rebase is in progress
	if err := utils.CheckRebaseInProgress(gctx); err != nil {
		return err
	}

	// Check if branch is empty when amending
	if !opts.CreateCommit {
		isEmpty, err := eng.IsBranchEmpty(gctx, currentBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch is empty: %w", err)
		}
		if isEmpty {
			// If branch is empty, we must create a new commit
			opts.CreateCommit = true
			splog.Info("Branch has no commits, creating new commit instead of amending.")
		}
	}

	// Stage changes based on flags
	stagingOpts := utils.StagingOptions{
		All:    opts.All,
		Update: opts.Update,
		Patch:  opts.Patch,
	}
	if err := utils.StageChanges(gctx, stagingOpts); err != nil {
		return err
	}

	commitMessage := opts.Message
	if commitMessage == "" && !utils.IsInteractive() && !opts.NoEdit {
		stdinMsg, err := utils.ReadFromStdin()
		if err == nil && stdinMsg != "" {
			commitMessage = stdinMsg
			opts.NoEdit = true
		}
	}

	// Check if there are staged changes (for new commits)
	if opts.CreateCommit {
		hasStagedChanges, err := git.HasStagedChanges(gctx)
		if err != nil {
			return fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStagedChanges {
			return fmt.Errorf("no staged changes to commit. Use -a to stage all changes, or stage changes manually with 'git add'")
		}
	}

	// Perform the commit
	commitOpts := git.CommitOptions{
		Amend:       !opts.CreateCommit,
		Message:     commitMessage,
		NoEdit:      opts.NoEdit,
		Edit:        opts.Edit,
		Verbose:     opts.Verbose,
		ResetAuthor: opts.ResetAuthor,
	}

	if err := git.CommitWithOptions(commitOpts); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Log success
	if opts.CreateCommit {
		splog.Info("Created new commit in %s.", tui.ColorBranchName(currentBranch, true))
	} else {
		splog.Info("Amended commit in %s.", tui.ColorBranchName(currentBranch, true))
	}

	// Restack upstack branches
	upstackBranches := eng.GetRelativeStackUpstack(currentBranch)

	if len(upstackBranches) > 0 {
		splog.Info("Restacking %d upstack branch(es)...", len(upstackBranches))
		if err := RestackBranches(gctx, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}

// interactiveRebaseAction performs an interactive rebase on the branch's commits
func interactiveRebaseAction(ctx *runtime.Context, _ ModifyOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	currentBranch := eng.CurrentBranch()

	// Get the parent branch to determine rebase base
	parent := eng.GetParent(currentBranch.Name)
	if parent == "" {
		parent = eng.Trunk().Name
	}

	splog.Info("Starting interactive rebase for %s onto %s...",
		tui.ColorBranchName(currentBranch.Name, true),
		tui.ColorBranchName(parent, false))

	// Run interactive rebase
	if err := git.RunGitCommandInteractive("rebase", "-i", parent); err != nil {
		// Check if rebase is in progress (conflict or user canceled)
		if git.IsRebaseInProgress(gctx) {
			return fmt.Errorf("interactive rebase paused. Resolve conflicts and run 'git rebase --continue' or 'git rebase --abort'")
		}
		// Rebase might have been aborted by user
		return nil
	}

	splog.Info("Interactive rebase completed.")

	// Restack upstack branches
	upstackBranches := eng.GetRelativeStackUpstack(currentBranch.Name)

	if len(upstackBranches) > 0 {
		splog.Info("Restacking %d upstack branch(es)...", len(upstackBranches))
		if err := RestackBranches(gctx, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
