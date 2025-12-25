package create

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the create command
type Options struct {
	BranchName    string
	Message       string
	Scope         string
	All           bool
	Insert        bool
	Patch         bool
	Update        bool
	Verbose       int
	BranchPattern config.BranchPattern
	// SelectedChildren is used to specify which children to move during insert
	// in non-interactive mode (mostly for tests)
	SelectedChildren []string
}

// Action creates a new branch stacked on top of the current branch
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch, err := utils.ValidateOnBranch(ctx.Engine)
	if err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	snapshotOpts := actions.NewSnapshot("create",
		actions.WithArg(opts.BranchName),
		actions.WithFlagValue("-m", opts.Message),
		actions.WithFlagValue("--scope", opts.Scope),
		actions.WithFlag(opts.All, "--all"),
		actions.WithFlag(opts.Insert, "--insert"),
		actions.WithFlag(opts.Patch, "--patch"),
		actions.WithFlag(opts.Update, "--update"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Handle staging first if we might need the message to name the branch
	hasStaged, err := git.HasStagedChanges(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	// Stage changes based on flags or prompt
	if opts.All || opts.Update || opts.Patch {
		stagingOpts := utils.StagingOptions{
			All:    opts.All,
			Update: opts.Update,
			Patch:  opts.Patch,
		}
		if err := utils.StageChanges(ctx.Context, stagingOpts); err != nil {
			return err
		}
		hasStaged = true
	} else if !hasStaged && utils.IsInteractive() {
		hasUnstaged, err := git.HasUnstagedChanges(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to check unstaged changes: %w", err)
		}

		if hasUnstaged {
			confirmed, err := tui.PromptConfirm("You have unstaged changes. Would you like to stage them?", false)
			if err == nil && confirmed {
				if err := git.StageAll(ctx.Context); err != nil {
					return fmt.Errorf("failed to stage changes: %w", err)
				}
				hasStaged = true
			}
		}
	}

	// Get commit message
	commitMessage := opts.Message
	if commitMessage == "" && !utils.IsInteractive() {
		stdinMsg, err := utils.ReadFromStdin()
		if err == nil && stdinMsg != "" {
			commitMessage = stdinMsg
		}
	}

	// Get commit message for branch name generation (if needed)
	commitMessage, err = getCommitMessageForBranch(ctx, &opts, commitMessage)
	if err != nil {
		return err
	}

	// Determine branch
	// Use provided scope if given, otherwise inherit from parent
	var scopeToUse string
	if opts.Scope != "" {
		scopeToUse = opts.Scope
	} else {
		parentScope := eng.GetScopeInternal(currentBranch)
		scopeToUse = parentScope.String()
	}
	branch, err := determineBranch(ctx, &opts, commitMessage, scopeToUse)
	if err != nil {
		return err
	}
	branchName := branch.Name

	// Check if branch already exists
	allBranches := eng.AllBranches()
	for _, existingBranch := range allBranches {
		if branch.Equal(existingBranch) {
			return fmt.Errorf("branch %s already exists", branchName)
		}
	}

	// Create and checkout new branch
	if err := git.CreateAndCheckoutBranch(ctx.Context, branch.Name); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Commit if there are staged changes
	if hasStaged {
		if err := git.Commit(commitMessage, opts.Verbose); err != nil {
			// Clean up branch on commit failure
			_ = git.DeleteBranch(ctx.Context, branch.Name)
			return fmt.Errorf("failed to commit: %w", err)
		}
	} else {
		splog.Info("No staged changes; created a branch with no commit.")
	}

	// Track the branch with current branch as parent
	if err := eng.TrackBranch(ctx.Context, branchName, currentBranch); err != nil {
		// Log error but don't fail - branch is created, just not tracked
		splog.Info("Warning: failed to track branch: %v", err)
	}

	// Set scope: use provided scope if given, otherwise let it inherit from parent naturally
	if opts.Scope != "" {
		// Set explicit scope if provided
		newScope := engine.NewScope(opts.Scope)
		if err := eng.SetScope(branch, newScope); err != nil {
			splog.Info("Warning: failed to set scope: %v", err)
		}
	}
	// If no scope provided, don't set anything - it will inherit from parent automatically

	// Handle insert logic
	if opts.Insert {
		if err := handleInsert(ctx.Context, branchName, currentBranch, ctx, &opts); err != nil {
			splog.Info("Warning: failed to insert branch: %v", err)
		}
	} else {
		// Check if current branch has children and show tip
		currentBranchObj := eng.GetBranch(currentBranch)
		children := currentBranchObj.GetChildren()
		siblings := []string{}
		for _, child := range children {
			if child.Name != branchName {
				siblings = append(siblings, child.Name)
			}
		}
		if len(siblings) > 0 {
			splog.Info("Tip: To insert a created branch into the middle of your stack, use the `--insert` flag.")
		}
	}

	return nil
}

func determineBranch(ctx *runtime.Context, opts *Options, commitMessage string, scope string) (engine.Branch, error) {
	branchName := opts.BranchName
	if branchName == "" {
		// Get pattern from options (always valid, default applied in GetBranchPattern)
		pattern := opts.BranchPattern

		// Generate branch name from pattern
		var err error
		branchName, err = pattern.GetBranchName(ctx.Context, commitMessage, scope)
		if err != nil {
			return engine.Branch{}, err
		}
	} else {
		// Sanitize provided branch name
		branchName = utils.SanitizeBranchName(branchName)
	}

	return ctx.Engine.GetBranch(branchName), nil
}
