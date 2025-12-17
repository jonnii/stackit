// Package actions provides high-level operations for managing stacked branches,
// including creating branches, submitting PRs, syncing, and restacking.
package actions

import (
	"fmt"
	"os"

	"stackit.dev/stackit/internal/branchutil"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
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
}

// CreateAction creates a new branch stacked on top of the current branch
func CreateAction(ctx *runtime.Context, opts CreateOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Determine branch name
	branchName := opts.BranchName
	if branchName == "" {
		if opts.Message == "" {
			return fmt.Errorf("must specify either a branch name or commit message")
		}
		branchName = branchutil.GenerateBranchNameFromMessage(opts.Message)
		if branchName == "" {
			return fmt.Errorf("failed to generate branch name from commit message")
		}
	} else {
		// Sanitize provided branch name
		branchName = branchutil.SanitizeBranchName(branchName)
	}

	// Check if branch already exists
	allBranches := eng.AllBranchNames()
	for _, name := range allBranches {
		if name == branchName {
			return fmt.Errorf("branch %s already exists", branchName)
		}
	}

	// Create and checkout new branch
	if err := git.CreateAndCheckoutBranch(branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Handle staging
	hasStaged, err := git.HasStagedChanges()
	if err != nil {
		// Clean up branch on error
		git.DeleteBranch(branchName)
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	// Stage changes based on flags
	if opts.All {
		if err := git.StageAll(); err != nil {
			git.DeleteBranch(branchName)
			return fmt.Errorf("failed to stage all changes: %w", err)
		}
		hasStaged = true
	} else if opts.Update {
		if err := git.StageTracked(); err != nil {
			git.DeleteBranch(branchName)
			return fmt.Errorf("failed to stage tracked changes: %w", err)
		}
		hasStaged = true
	} else if opts.Patch {
		if err := git.StagePatch(); err != nil {
			git.DeleteBranch(branchName)
			return fmt.Errorf("failed to stage patch: %w", err)
		}
		hasStaged = true
	} else {
		// Check for unstaged changes and prompt if interactive
		hasUnstaged, err := git.HasUnstagedChanges()
		if err != nil {
			git.DeleteBranch(branchName)
			return fmt.Errorf("failed to check unstaged changes: %w", err)
		}

		if hasUnstaged && !hasStaged {
			// Check if we're in an interactive terminal
			if isInteractive() {
				ctx.Splog.Info("You have unstaged changes. Would you like to stage them? (y/n): ")
				var response string
				fmt.Scanln(&response)
				if response == "y" || response == "Y" || response == "yes" {
					if err := git.StageAll(); err != nil {
						git.DeleteBranch(branchName)
						return fmt.Errorf("failed to stage changes: %w", err)
					}
					hasStaged = true
				}
			}
		}
	}

	// Commit if there are staged changes
	if hasStaged {
		commitMessage := opts.Message
		if commitMessage == "" {
			// Try to get message from environment or prompt
			commitMessage = getCommitMessage()
		}

		if err := git.Commit(commitMessage, opts.Verbose); err != nil {
			// Clean up branch on commit failure
			git.DeleteBranch(branchName)
			return fmt.Errorf("failed to commit: %w", err)
		}
	} else {
		splog.Info("No staged changes; created a branch with no commit.")
	}

	// Track the branch with current branch as parent
	if err := eng.TrackBranch(branchName, currentBranch); err != nil {
		// Log error but don't fail - branch is created, just not tracked
		splog.Info("Warning: failed to track branch: %v", err)
	}

	// Handle insert logic
	if opts.Insert {
		if err := handleInsert(branchName, currentBranch, ctx); err != nil {
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
func handleInsert(newBranch, currentBranch string, ctx *runtime.Context) error {
	children := ctx.Engine.GetChildren(currentBranch)
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
	if len(siblings) > 1 && isInteractive() {
		ctx.Splog.Info("Current branch has multiple children. Select which should be moved onto the new branch:")
		for i, child := range siblings {
			ctx.Splog.Info("%d. %s", i+1, child)
		}
		ctx.Splog.Info("Enter numbers separated by commas (or 'all' for all): ")
		var response string
		fmt.Scanln(&response)

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
		if err := ctx.Engine.TrackBranch(child, newBranch); err != nil {
			return fmt.Errorf("failed to update parent for %s: %w", child, err)
		}
	}

	return nil
}

// isInteractive checks if we're in an interactive terminal
func isInteractive() bool {
	// Allow forcing non-interactive mode via environment variable
	// This is useful for tests and CI environments
	if os.Getenv("STACKIT_NON_INTERACTIVE") != "" {
		return false
	}

	// Check if stdin is a terminal
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// getCommitMessage gets commit message from environment or prompts user
func getCommitMessage() string {
	// Try GIT_EDITOR or fallback to prompting
	// For now, return empty and let git handle it
	return ""
}
