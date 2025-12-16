package actions

import (
	"fmt"
	"os"
	"os/exec"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
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

	// Dependencies
	Engine   engine.Engine
	Splog    *output.Splog
	RepoRoot string
}

// ModifyAction performs the modify operation
func ModifyAction(opts ModifyOptions) error {
	currentBranch := opts.Engine.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Check if we're on trunk
	if opts.Engine.IsTrunk(currentBranch) {
		return fmt.Errorf("cannot modify trunk branch %s", currentBranch)
	}

	// Handle interactive rebase separately
	if opts.InteractiveRebase {
		return interactiveRebaseAction(opts)
	}

	// Check if rebase is in progress
	if git.IsRebaseInProgress() {
		return fmt.Errorf("cannot modify while a rebase is in progress. Run 'stackit continue' or 'stackit abort'")
	}

	// Check if branch is empty when amending
	if !opts.CreateCommit {
		isEmpty, err := opts.Engine.IsBranchEmpty(currentBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch is empty: %w", err)
		}
		if isEmpty {
			// If branch is empty, we must create a new commit
			opts.CreateCommit = true
			opts.Splog.Info("Branch has no commits, creating new commit instead of amending.")
		}
	}

	// Stage changes based on flags
	if err := stageChanges(opts); err != nil {
		return err
	}

	// Check if there are staged changes (for new commits)
	if opts.CreateCommit {
		hasStagedChanges, err := git.HasStagedChanges()
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
		Message:     opts.Message,
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
		opts.Splog.Info("Created new commit in %s.", output.ColorBranchName(currentBranch, true))
	} else {
		opts.Splog.Info("Amended commit in %s.", output.ColorBranchName(currentBranch, true))
	}

	// Restack upstack branches
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := opts.Engine.GetRelativeStack(currentBranch, scope)

	if len(upstackBranches) > 0 {
		opts.Splog.Info("Restacking %d upstack branch(es)...", len(upstackBranches))
		if err := RestackBranches(upstackBranches, opts.Engine, opts.Splog, opts.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}

// stageChanges stages changes based on the options
func stageChanges(opts ModifyOptions) error {
	// Handle interactive patch staging first (takes precedence)
	if opts.Patch && !opts.All {
		return runInteractivePatch()
	}

	// Stage all changes (including untracked)
	if opts.All {
		return git.StageAll()
	}

	// Stage only tracked files
	if opts.Update {
		return git.StageTracked()
	}

	return nil
}

// runInteractivePatch runs git add -p interactively
func runInteractivePatch() error {
	cmd := exec.Command("git", "add", "-p")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("interactive patch staging failed: %w", err)
	}
	return nil
}

// interactiveRebaseAction performs an interactive rebase on the branch's commits
func interactiveRebaseAction(opts ModifyOptions) error {
	currentBranch := opts.Engine.CurrentBranch()

	// Get the parent branch to determine rebase base
	parent := opts.Engine.GetParent(currentBranch)
	if parent == "" {
		parent = opts.Engine.Trunk()
	}

	opts.Splog.Info("Starting interactive rebase for %s onto %s...",
		output.ColorBranchName(currentBranch, true),
		output.ColorBranchName(parent, false))

	// Run interactive rebase
	cmd := exec.Command("git", "rebase", "-i", parent)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if rebase is in progress (conflict or user cancelled)
		if git.IsRebaseInProgress() {
			return fmt.Errorf("interactive rebase paused. Resolve conflicts and run 'git rebase --continue' or 'git rebase --abort'")
		}
		// Rebase might have been aborted by user
		return nil
	}

	opts.Splog.Info("Interactive rebase completed.")

	// Restack upstack branches
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := opts.Engine.GetRelativeStack(currentBranch, scope)

	if len(upstackBranches) > 0 {
		opts.Splog.Info("Restacking %d upstack branch(es)...", len(upstackBranches))
		if err := RestackBranches(upstackBranches, opts.Engine, opts.Splog, opts.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
