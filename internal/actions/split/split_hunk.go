package split

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// splitByHunkEngine is a minimal interface needed for splitting by hunk
type splitByHunkEngine interface {
	engine.BranchReader
	engine.PRManager
	engine.SplitManager
}

// splitByHunk splits a branch by interactively staging hunks.
//
// Algorithm:
//  1. Detach HEAD and soft reset to the parent branch's tip, leaving changes unstaged.
//  2. Loop until no unstaged changes remain:
//     a. Show remaining unstaged changes.
//     b. Interactively prompt the user to stage hunks for the next branch.
//     c. Prompt for a commit message and branch name.
//     d. Create a new commit with the staged changes.
//  3. Return the created branch names.
func splitByHunk(ctx context.Context, branchToSplit string, eng splitByHunkEngine, splog *tui.Splog) (*Result, error) {
	// Detach and reset branch changes
	if err := eng.DetachAndResetBranchChanges(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to detach and reset: %w", err)
	}

	branchNames := []string{}

	// Get default commit message
	commitMessages, err := eng.GetAllCommits(branchToSplit, engine.CommitFormatMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	defaultCommitMessage := strings.Join(commitMessages, "\n\n")

	// Show instructions
	splog.Info("Splitting %s into multiple single-commit branches.", tui.ColorBranchName(branchToSplit, true))
	prInfo, _ := eng.GetPrInfo(branchToSplit)
	if prInfo != nil && prInfo.Number != nil {
		splog.Info("If any of the new branches keeps the name %s, it will be linked to PR #%d.",
			tui.ColorBranchName(branchToSplit, true), *prInfo.Number)
	}
	splog.Info("")
	splog.Info("For each branch you'd like to create:")
	splog.Info("1. Follow the prompts to stage the changes that you'd like to include.")
	splog.Info("2. Enter a commit message.")
	splog.Info("3. Pick a branch name.")
	splog.Info("The command will continue until all changes have been added to a new branch.")
	splog.Info("")

	// Loop while there are unstaged changes
	for {
		hasUnstaged, err := git.HasUnstagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check unstaged changes: %w", err)
		}
		if !hasUnstaged {
			break
		}

		// Show remaining changes
		unstagedDiff, err := git.GetUnstagedDiff(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get unstaged diff: %w", err)
		}
		splog.Info("Remaining changes:")
		splog.Info("  %s", strings.ReplaceAll(unstagedDiff, "\n", "\n  "))
		splog.Info("")

		splog.Info("Stage changes for branch %d:", len(branchNames)+1)

		// Stage patch interactively
		if err := git.StagePatch(); err != nil {
			// If user cancels, restore branch
			_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
			return nil, fmt.Errorf("canceled: no new branches created")
		}

		// Check if anything was staged
		hasStaged, err := git.HasStagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStaged {
			splog.Info("No changes staged, skipping this branch.")
			continue
		}

		// Commit with message
		commitMessage := defaultCommitMessage
		var editMessage bool
		prompt := &survey.Confirm{
			Message: "Edit commit message?",
			Default: true,
		}
		if err := survey.AskOne(prompt, &editMessage); err != nil {
			return nil, fmt.Errorf("canceled")
		}

		if editMessage {
			// Get message from user
			msg, err := tui.OpenEditor(defaultCommitMessage, "COMMIT_EDITMSG-*")
			if err != nil {
				return nil, err
			}
			commitMessage = utils.CleanCommitMessage(msg)
		}

		// Create commit
		if err := git.Commit(commitMessage, 0); err != nil {
			return nil, fmt.Errorf("failed to create commit: %w", err)
		}

		// Get branch name
		branchName, err := promptBranchName(branchNames, branchToSplit, len(branchNames)+1, eng)
		if err != nil {
			return nil, err
		}
		branchNames = append(branchNames, branchName)
	}

	return &Result{
		BranchNames:  branchNames,
		BranchPoints: makeRange(len(branchNames)), // Each branch is a single commit
	}, nil
}
