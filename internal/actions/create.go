// Package actions provides high-level operations for managing stacked branches,
// including creating branches, submitting PRs, syncing, and restacking.
package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
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
	AI         bool
	AIClient   ai.AIClient
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

	// If AI is enabled and no branch name/message provided, generate commit message first
	commitMessage := opts.Message
	if opts.AI && opts.AIClient != nil && opts.BranchName == "" && commitMessage == "" {
		// Stage changes first to get diff for AI
		// Check if we have any changes to stage
		hasUnstaged, err := git.HasUnstagedChanges(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to check unstaged changes: %w", err)
		}

		hasUntracked, err := git.HasUntrackedFiles(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to check untracked files: %w", err)
		}

		hasStaged, err := git.HasStagedChanges(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to check staged changes: %w", err)
		}

		if !hasStaged && !hasUnstaged && !hasUntracked {
			return fmt.Errorf("no changes to commit. Stage some changes first or provide a branch name or commit message")
		}

		// Stage changes if needed
		if !hasStaged && hasUnstaged {
			// Auto-stage all changes for AI generation
			if err := git.StageAll(ctx.Context); err != nil {
				return fmt.Errorf("failed to stage changes for AI: %w", err)
			}
			splog.Debug("Staged changes for AI commit message generation")
		}

		// Get staged diff
		diff, err := git.GetStagedDiff(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to get staged diff: %w", err)
		}

		if diff == "" {
			return fmt.Errorf("no staged changes found. Stage some changes first or provide a branch name or commit message")
		}

		// Generate commit message using AI
		splog.Debug("Generating commit message using AI from staged changes")
		generatedMessage, err := opts.AIClient.GenerateCommitMessage(ctx.Context, diff)
		if err != nil {
			return fmt.Errorf("failed to generate commit message with AI: %w", err)
		}

		if generatedMessage == "" {
			return fmt.Errorf("AI generated empty commit message")
		}

		commitMessage = generatedMessage
		splog.Debug("AI generated commit message: %s", commitMessage)
	}

	// Determine branch name
	branchName := opts.BranchName
	if branchName == "" {
		if commitMessage == "" {
			if opts.AI && opts.AIClient != nil {
				return fmt.Errorf("must specify either a branch name, commit message, or use --ai with staged changes")
			}
			return fmt.Errorf("must specify either a branch name or commit message")
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

	// Handle staging (skip if AI already staged changes)
	var hasStaged bool
	var alreadyStagedByAI bool
	if opts.AI && opts.AIClient != nil && opts.BranchName == "" && opts.Message == "" {
		// AI already staged changes, check if we have staged changes
		hasStaged, err = git.HasStagedChanges(ctx.Context)
		if err != nil {
			_ = git.DeleteBranch(ctx.Context, branchName)
			return fmt.Errorf("failed to check staged changes: %w", err)
		}
		alreadyStagedByAI = hasStaged
	}

	if !alreadyStagedByAI {
		// Check staged changes
		_, err = git.HasStagedChanges(ctx.Context)
		if err != nil {
			// Clean up branch on error
			_ = git.DeleteBranch(ctx.Context, branchName)
			return fmt.Errorf("failed to check staged changes: %w", err)
		}

		// Stage changes based on flags
		stagingOpts := utils.StagingOptions{
			All:    opts.All,
			Update: opts.Update,
			Patch:  opts.Patch,
		}
		if err := utils.StageChanges(ctx.Context, stagingOpts); err != nil {
			_ = git.DeleteBranch(ctx.Context, branchName)
			return err
		}

		// If no staging flags, check for unstaged changes and prompt if interactive
		if !opts.All && !opts.Update && !opts.Patch {
			hasStaged, err = git.HasStagedChanges(ctx.Context)
			if err != nil {
				_ = git.DeleteBranch(ctx.Context, branchName)
				return fmt.Errorf("failed to check staged changes: %w", err)
			}

			hasUnstaged, err := git.HasUnstagedChanges(ctx.Context)
			if err != nil {
				_ = git.DeleteBranch(ctx.Context, branchName)
				return fmt.Errorf("failed to check unstaged changes: %w", err)
			}

			if hasUnstaged && !hasStaged {
				// Check if we're in an interactive terminal
				if utils.IsInteractive() {
					ctx.Splog.Info("You have unstaged changes. Would you like to stage them? (y/n): ")
					var response string
					_, _ = fmt.Scanln(&response)
					if response == "y" || response == "Y" || response == yesResponse {
						if err := git.StageAll(ctx.Context); err != nil {
							_ = git.DeleteBranch(ctx.Context, branchName)
							return fmt.Errorf("failed to stage changes: %w", err)
						}
						hasStaged = true
					}
				}
			}
		} else {
			hasStaged = true
		}
	}

	// Commit if there are staged changes
	if hasStaged {
		if commitMessage == "" {
			// Try to get message from environment or prompt
			commitMessage = getCommitMessage()
		}

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
		if err := handleInsert(ctx.Context, branchName, currentBranch, ctx); err != nil {
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

const yesResponse = "yes"

// handleInsert moves children of the current branch to be children of the new branch
func handleInsert(ctx context.Context, newBranch, currentBranch string, runtimeCtx *runtime.Context) error {
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
	if len(siblings) > 1 && utils.IsInteractive() {
		runtimeCtx.Splog.Info("Current branch has multiple children. Select which should be moved onto the new branch:")
		for i, child := range siblings {
			runtimeCtx.Splog.Info("%d. %s", i+1, child)
		}
		runtimeCtx.Splog.Info("Enter numbers separated by commas (or 'all' for all): ")
		var response string
		_, _ = fmt.Scanln(&response)

		if response == "all" || response == "All" || response == "ALL" {
			toMove = siblings
		} else {
			// Parse comma-separated numbers
			// For now, just move all - proper parsing can be added later
			toMove = siblings
		}
	} else {
		// Single child or non-interactive - move all
		toMove = siblings
	}

	// Update parent for each child to move
	for _, child := range toMove {
		if err := runtimeCtx.Engine.TrackBranch(ctx, child, newBranch); err != nil {
			return fmt.Errorf("failed to update parent for %s: %w", child, err)
		}
	}

	return nil
}

// getCommitMessage gets commit message from environment or prompts user
func getCommitMessage() string {
	// Try GIT_EDITOR or fallback to prompting
	// For now, return empty and let git handle it
	return ""
}
