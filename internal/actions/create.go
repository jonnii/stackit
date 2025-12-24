// Package actions provides high-level operations for managing stacked branches,
// including creating branches, submitting PRs, syncing, and restacking.
package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// CreateOptions contains options for the create command
type CreateOptions struct {
	BranchName    string
	Message       string
	Scope         string // Optional scope for the new branch
	All           bool
	Insert        bool
	Patch         bool
	Update        bool
	Verbose       int
	BranchPattern config.BranchPattern // Branch name pattern from config
	// SelectedChildren is used to specify which children to move during insert
	// in non-interactive mode (mostly for tests)
	SelectedChildren []string
}

// CreateAction creates a new branch stacked on top of the current branch
func CreateAction(ctx *runtime.Context, opts CreateOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch, err := utils.ValidateOnBranch(ctx)
	if err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	snapshotOpts := NewSnapshot("create",
		WithArg(opts.BranchName),
		WithFlagValue("-m", opts.Message),
		WithFlagValue("--scope", opts.Scope),
		WithFlag(opts.All, "--all"),
		WithFlag(opts.Insert, "--insert"),
		WithFlag(opts.Patch, "--patch"),
		WithFlag(opts.Update, "--update"),
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
	commitMessage, err = getCommitMessageForBranch(ctx, opts, commitMessage)
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
	branch, err := determineBranch(ctx, opts, commitMessage, scopeToUse)
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
		if err := handleInsert(ctx.Context, branchName, currentBranch, ctx, opts); err != nil {
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

// handleInsert moves children of the current branch to be children of the new branch
func handleInsert(ctx context.Context, newBranch, currentBranch string, runtimeCtx *runtime.Context, opts CreateOptions) error {
	currentBranchObj := runtimeCtx.Engine.GetBranch(currentBranch)
	children := currentBranchObj.GetChildren()
	siblings := []string{}
	for _, child := range children {
		if child.Name != newBranch {
			siblings = append(siblings, child.Name)
		}
	}

	if len(siblings) == 0 {
		return nil
	}

	// If multiple children, prompt user to select which to move
	var toMove []string
	switch {
	case len(opts.SelectedChildren) > 0:
		// Use pre-selected children (for tests)
		for _, selected := range opts.SelectedChildren {
			for _, sibling := range siblings {
				if selected == sibling {
					toMove = append(toMove, sibling)
					break
				}
			}
		}
	case len(siblings) > 1 && utils.IsInteractive():
		runtimeCtx.Splog.Info("Current branch has multiple children. Select which should be moved onto the new branch:")
		options := []tui.SelectOption{
			{Label: "All children", Value: "all"},
		}
		for _, child := range siblings {
			options = append(options, tui.SelectOption{Label: child, Value: child})
		}

		selected, err := tui.PromptSelect("Which child should be moved onto the new branch?", options, 0)
		if err != nil {
			return err
		}

		if selected == "all" {
			toMove = siblings
		} else {
			toMove = []string{selected}
		}
	default:
		// Single child or non-interactive - move all
		toMove = siblings
	}

	// Update parent for each child to move
	for _, child := range toMove {
		if err := runtimeCtx.Engine.TrackBranch(ctx, child, newBranch); err != nil {
			return fmt.Errorf("failed to update parent for %s: %w", child, err)
		}

		// Restack the child onto the new branch to physically insert it
		childBranch := runtimeCtx.Engine.GetBranch(child)
		res, err := runtimeCtx.Engine.RestackBranch(ctx, childBranch)
		if err != nil {
			runtimeCtx.Splog.Info("Warning: failed to restack %s onto %s: %v", child, newBranch, err)
			continue
		}
		if res.Result == engine.RestackConflict {
			runtimeCtx.Splog.Info("Conflict restacking %s onto %s. Please resolve manually or run 'stackit sync --restack'.", child, newBranch)
		} else if res.Result == engine.RestackDone {
			runtimeCtx.Splog.Info("Restacked %s onto %s.", child, newBranch)
		}
	}

	return nil
}

// getCommitMessage gets commit message from editor
func getCommitMessage(ctx context.Context) (string, error) {
	template, err := git.GetCommitTemplate(ctx)
	if err != nil {
		return "", err
	}

	msg, err := tui.OpenEditor(template, "COMMIT_EDITMSG-*")
	if err != nil {
		return "", err
	}

	return utils.CleanCommitMessage(msg), nil
}

// getCommitMessageForBranch gets the commit message needed for branch name generation.
// If branch name is not provided and commit message is empty, it will prompt for one in interactive mode.
func getCommitMessageForBranch(ctx *runtime.Context, opts CreateOptions, commitMessage string) (string, error) {
	// If branch name is provided, we don't need commit message for branch generation
	if opts.BranchName != "" {
		return commitMessage, nil
	}

	// If commit message is empty, we need to get it
	if commitMessage == "" {
		if !utils.IsInteractive() {
			return "", fmt.Errorf("must specify either a branch name or commit message")
		}

		// Interactive: get commit message from editor
		msg, err := getCommitMessage(ctx.Context)
		if err != nil {
			return "", err
		}
		if msg == "" {
			return "", fmt.Errorf("aborting due to empty commit message")
		}
		return msg, nil
	}

	return commitMessage, nil
}

// determineBranch determines the branch name and returns a Branch object.
// It expects a clean commit message to be provided if branch name is not specified.
func determineBranch(ctx *runtime.Context, opts CreateOptions, commitMessage string, scope string) (engine.Branch, error) {
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
