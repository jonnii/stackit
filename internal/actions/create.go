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
	BranchName string
	Message    string
	All        bool
	Insert     bool
	Patch      bool
	Update     bool
	Verbose    int
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
	args := []string{}
	if opts.BranchName != "" {
		args = append(args, opts.BranchName)
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	if opts.All {
		args = append(args, "--all")
	}
	if opts.Insert {
		args = append(args, "--insert")
	}
	if opts.Patch {
		args = append(args, "--patch")
	}
	if opts.Update {
		args = append(args, "--update")
	}
	if err := eng.TakeSnapshot("create", args); err != nil {
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

	// Determine branch name
	branchName := opts.BranchName
	if branchName == "" {
		if commitMessage == "" {
			if !utils.IsInteractive() {
				return fmt.Errorf("must specify either a branch name or commit message")
			}

			// Interactive: get commit message from editor
			msg, err := getCommitMessage(ctx.Context)
			if err != nil {
				return err
			}
			if msg == "" {
				return fmt.Errorf("aborting due to empty commit message")
			}
			commitMessage = msg
		}

		// Get pattern from config
		pattern, err := config.GetBranchNamePattern(ctx.RepoRoot)
		if err != nil {
			return fmt.Errorf("failed to get branch name pattern: %w", err)
		}

		// Get username and date for pattern processing
		username, err := git.GetUserName(ctx.Context)
		if err != nil {
			// If we can't get username, use empty string (will be sanitized)
			username = ""
		}
		date := git.GetCurrentDate()

		// Process the pattern
		branchName = utils.ProcessBranchNamePattern(pattern, username, date, commitMessage)
		if branchName == "" {
			return fmt.Errorf("failed to generate branch name from commit message")
		}
	} else {
		// Sanitize provided branch name
		branchName = utils.SanitizeBranchName(branchName)
	}

	// Check if branch already exists
	allBranches := eng.AllBranchNames()
	for _, name := range allBranches {
		if name == branchName {
			return fmt.Errorf("branch %s already exists", branchName)
		}
	}

	// Create and checkout new branch
	if err := git.CreateAndCheckoutBranch(ctx.Context, branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Commit if there are staged changes
	if hasStaged {
		if err := git.Commit(commitMessage, opts.Verbose); err != nil {
			// Clean up branch on commit failure
			_ = git.DeleteBranch(ctx.Context, branchName)
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

	// Handle insert logic
	if opts.Insert {
		if err := handleInsert(ctx.Context, branchName, currentBranch, ctx, opts); err != nil {
			splog.Info("Warning: failed to insert branch: %v", err)
		}
	} else {
		// Check if current branch has children and show tip
		children := eng.GetChildren(currentBranch)
		siblings := []string{}
		for _, child := range children {
			if child != branchName {
				siblings = append(siblings, child)
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
	children := runtimeCtx.Engine.GetChildren(currentBranch)
	siblings := []string{}
	for _, child := range children {
		if child != newBranch {
			siblings = append(siblings, child)
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
		res, err := runtimeCtx.Engine.RestackBranch(ctx, child)
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
