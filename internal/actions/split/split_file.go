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

// splitByFileEngine is a minimal interface needed for splitting by file
type splitByFileEngine interface {
	engine.BranchReader
	engine.BranchWriter
}

// splitByFile splits a branch by extracting specified files to a new parent branch.
//
// Algorithm:
//  1. Determine the parent of the branch to split.
//  2. Create a new "split" branch from the parent.
//  3. Checkout the specified files from the original branch into the new split branch.
//  4. Commit the extracted files on the new split branch.
//  5. Checkout the original branch and remove the extracted files.
//  6. Commit the removals on the original branch.
//  7. Update the original branch's parent to be the new split branch.
func splitByFile(ctx context.Context, branchToSplit string, pathspecs []string, eng splitByFileEngine) (*Result, error) {
	// Get parent branch
	parentBranchName := eng.GetParentPrecondition(branchToSplit)

	// Generate new branch name
	newBranchName := branchToSplit + "_split"
	allBranches := eng.AllBranchNames()
	for utils.ContainsString(allBranches, newBranchName) {
		newBranchName += "_split"
	}

	// First checkout the parent branch so the new branch starts from there
	if err := git.CheckoutBranch(ctx, parentBranchName); err != nil {
		return nil, fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Create new branch from parent
	if err := git.CreateAndCheckoutBranch(ctx, newBranchName); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout files from branchToSplit
	args := append([]string{"checkout", branchToSplit, "--"}, pathspecs...)
	if _, err := git.RunGitCommandWithContext(ctx, args...); err != nil {
		// Cleanup: delete the new branch
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to checkout files: %w", err)
	}

	// Stage all changes
	if err := git.StageAll(ctx); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit
	commitMessage := fmt.Sprintf("Extract %s from %s", strings.Join(pathspecs, ", "), branchToSplit)
	if err := git.Commit(commitMessage, 0); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Track the new branch
	if err := eng.TrackBranch(ctx, newBranchName, parentBranchName); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to track branch: %w", err)
	}

	// Checkout original branch and remove the files
	if err := git.CheckoutBranch(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to checkout original branch: %w", err)
	}

	// Remove the files from the original branch (both index and working directory)
	args = append([]string{"rm"}, pathspecs...)
	if _, err := git.RunGitCommandWithContext(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to remove files: %w", err)
	}

	// Commit the removal
	commitMessage = fmt.Sprintf("Remove %s (moved to %s)", strings.Join(pathspecs, ", "), newBranchName)
	if err := git.Commit(commitMessage, 0); err != nil {
		return nil, fmt.Errorf("failed to commit removal: %w", err)
	}

	// Update original branch's parent to be the new split branch
	// This creates the hierarchy: parent -> newBranch -> originalBranch
	if err := eng.SetParent(ctx, branchToSplit, newBranchName); err != nil {
		return nil, fmt.Errorf("failed to update parent: %w", err)
	}

	return &Result{
		BranchNames:  []string{newBranchName},
		BranchPoints: []int{0}, // Single commit
	}, nil
}

// promptForFiles shows an interactive file selector for split --by-file
func promptForFiles(ctx context.Context, branchToSplit string, eng engine.BranchReader, splog *tui.Splog) ([]string, error) {
	// Get the parent branch to compare against
	parentBranchName := eng.GetParentPrecondition(branchToSplit)

	// Get merge base between branch and parent
	mergeBase, err := git.GetMergeBase(ctx, branchToSplit, parentBranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get list of changed files
	changedFiles, err := git.GetChangedFiles(ctx, mergeBase, branchToSplit)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		return nil, fmt.Errorf("no files changed in branch %s", branchToSplit)
	}

	if len(changedFiles) == 1 {
		return nil, fmt.Errorf("only one file changed in branch - nothing to split")
	}

	// Show instructions
	splog.Info("Splitting %s by file.", tui.ColorBranchName(branchToSplit, true))
	splog.Info("Select the files to extract to a new parent branch.")
	splog.Info("The remaining files will stay on %s.", tui.ColorBranchName(branchToSplit, true))
	splog.Info("")

	// Prompt for file selection
	var selectedFiles []string
	prompt := &survey.MultiSelect{
		Message: "Select files to extract:",
		Options: changedFiles,
	}
	if err := survey.AskOne(prompt, &selectedFiles); err != nil {
		return nil, fmt.Errorf("canceled")
	}

	// Validate that not all files were selected
	if len(selectedFiles) == len(changedFiles) {
		return nil, fmt.Errorf("cannot extract all files - at least one must remain on the original branch")
	}

	return selectedFiles, nil
}
