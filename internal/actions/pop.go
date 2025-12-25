package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// PopOptions contains options for the pop command
type PopOptions struct {
	// Currently no options, but structure is here for future extensibility
}

// PopAction deletes the current branch but retains the state of files in the working tree
func PopAction(ctx *runtime.Context, _ PopOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Validate we're on a branch
	currentBranch, err := utils.ValidateOnBranch(ctx.Engine)
	if err != nil {
		return err
	}

	// Check if on trunk
	currentBranchObj := eng.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return fmt.Errorf("cannot pop trunk branch")
	}

	// Check if branch is tracked
	if !currentBranchObj.IsTracked() {
		return fmt.Errorf("cannot pop untracked branch %s", currentBranch)
	}

	// Check if rebase is in progress
	if err := utils.CheckRebaseInProgress(ctx.Context); err != nil {
		return err
	}

	// Check for uncommitted changes
	if utils.HasUncommittedChanges(ctx.Context) {
		return fmt.Errorf("cannot pop with uncommitted changes. Please commit or stash them first")
	}

	// Get parent branch
	// currentBranchObj already declared above
	parent := eng.GetParent(currentBranchObj)
	parentName := ""
	if parent == nil {
		parentName = eng.Trunk().Name
	} else {
		parentName = parent.Name
	}

	// Get parent branch revision
	parentBranch := eng.GetBranch(parentName)
	parentRev, err := parentBranch.GetRevision()
	if err != nil {
		return fmt.Errorf("failed to get parent revision: %w", err)
	}

	// Soft reset to parent - this uncommits the current branch's changes
	// and stages them, keeping the working tree unchanged
	if err := git.SoftReset(ctx.Context, parentRev); err != nil {
		return fmt.Errorf("failed to reset to parent: %w", err)
	}

	// Checkout parent branch
	if err := git.CheckoutBranch(ctx.Context, parentBranch.Name); err != nil {
		return fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Delete the old branch (this will also reparent any children)
	if err := eng.DeleteBranch(ctx.Context, currentBranchObj.Name); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// Check how many changes are staged
	hasStaged, err := git.HasStagedChanges(ctx.Context)
	if err == nil && hasStaged {
		splog.Info("Popped branch %s. Changes are now staged on %s.",
			tui.ColorBranchName(currentBranch, false),
			tui.ColorBranchName(parentName, false))
	} else {
		splog.Info("Popped branch %s. Switched to %s.",
			tui.ColorBranchName(currentBranch, false),
			tui.ColorBranchName(parentName, false))
	}

	return nil
}
