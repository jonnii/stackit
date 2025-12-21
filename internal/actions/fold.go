package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// FoldOptions contains options for the fold command
type FoldOptions struct {
	Keep bool // If true, keeps the name of the current branch instead of using the name of its parent
}

// FoldAction performs the fold operation
func FoldAction(ctx *runtime.Context, opts FoldOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Validate we're on a branch
	currentBranch, err := utils.ValidateOnBranch(ctx)
	if err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	args := []string{}
	if opts.Keep {
		args = append(args, "--keep")
	}
	if err := eng.TakeSnapshot(gctx, "fold", args); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Check if on trunk
	if eng.IsTrunk(currentBranch) {
		return fmt.Errorf("cannot fold trunk branch")
	}

	// Check if branch is tracked
	if !eng.IsBranchTracked(currentBranch) {
		return fmt.Errorf("cannot fold untracked branch %s", currentBranch)
	}

	// Check if rebase is in progress
	if err := utils.CheckRebaseInProgress(gctx); err != nil {
		return err
	}

	// Check for uncommitted changes
	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("cannot fold with uncommitted changes. Please commit or stash them first")
	}

	// Get parent branch
	parent := eng.GetParent(currentBranch)
	if parent == "" {
		parent = eng.Trunk()
	}

	if opts.Keep {
		// Prevent folding onto trunk with --keep, as that would delete trunk
		if eng.IsTrunk(parent) {
			return fmt.Errorf("cannot fold into trunk with --keep because it would delete the trunk branch")
		}
		return foldWithKeep(gctx, ctx, currentBranch, parent, eng, splog)
	}

	return foldNormal(gctx, ctx, currentBranch, parent, eng, splog)
}

// foldNormal performs a normal fold: merge current branch into parent, then delete current branch
func foldNormal(gctx context.Context, ctx *runtime.Context, currentBranch, parent string, eng engine.Engine, splog *tui.Splog) error {
	// Checkout parent branch
	if err := git.CheckoutBranch(gctx, parent); err != nil {
		return fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Rebuild engine so it knows we're on the parent branch
	if err := eng.Rebuild(gctx, eng.Trunk()); err != nil {
		return fmt.Errorf("failed to rebuild engine: %w", err)
	}

	// Try fast-forward merge first, fallback to regular merge
	_, err := git.RunGitCommandWithContext(gctx, "merge", "--ff-only", currentBranch)
	if err != nil {
		// Fast-forward failed, try regular merge
		_, err = git.RunGitCommandWithContext(gctx, "merge", "--no-edit", currentBranch)
		if err != nil {
			return fmt.Errorf("failed to merge %s into %s due to conflicts. Please resolve the conflicts and run 'git commit', or abort with 'git merge --abort'", currentBranch, parent)
		}
	}

	// Get all descendants of parent before deletion (for restacking)
	descendants := eng.GetRelativeStack(parent, engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    false,
		RecursiveParents:  false,
	})

	// Delete the current branch (this will automatically reparent its children to parent)
	if err := eng.DeleteBranch(gctx, currentBranch); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	splog.Info("Folded %s into %s.",
		tui.ColorBranchName(currentBranch, true),
		tui.ColorBranchName(parent, false))

	// Restack all descendants of the parent
	if len(descendants) > 0 {
		// Rebuild engine to reflect the deletion
		if err := eng.Rebuild(gctx, eng.Trunk()); err != nil {
			return fmt.Errorf("failed to rebuild engine: %w", err)
		}

		// Get updated descendants list (current branch's children are now children of parent)
		updatedDescendants := eng.GetRelativeStack(parent, engine.Scope{
			RecursiveChildren: true,
			IncludeCurrent:    false,
			RecursiveParents:  false,
		})

		if err := RestackBranches(gctx, updatedDescendants, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}

// foldWithKeep performs a fold with --keep: merge parent into current branch, then delete parent
func foldWithKeep(gctx context.Context, ctx *runtime.Context, currentBranch, parent string, eng engine.Engine, splog *tui.Splog) error {
	// Get all children of parent (siblings + current branch)
	allChildren := eng.GetChildren(parent)

	// Identify siblings (children of parent excluding current branch)
	siblings := []string{}
	for _, child := range allChildren {
		if child != currentBranch {
			siblings = append(siblings, child)
		}
	}

	// Ensure we're on the current branch
	if err := git.CheckoutBranch(gctx, currentBranch); err != nil {
		return fmt.Errorf("failed to checkout current branch: %w", err)
	}

	// Try fast-forward merge first, fallback to regular merge
	_, err := git.RunGitCommandWithContext(gctx, "merge", "--ff-only", parent)
	if err != nil {
		// Fast-forward failed, try regular merge
		_, err = git.RunGitCommandWithContext(gctx, "merge", "--no-edit", parent)
		if err != nil {
			return fmt.Errorf("failed to merge %s into %s due to conflicts. Please resolve the conflicts and run 'git commit', or abort with 'git merge --abort'", parent, currentBranch)
		}
	}

	// Delete the parent branch (this will reparent current branch and siblings to grandparent)
	if err := eng.DeleteBranch(gctx, parent); err != nil {
		return fmt.Errorf("failed to delete parent branch: %w", err)
	}

	// Rebuild engine to reflect the deletion
	if err := eng.Rebuild(gctx, eng.Trunk()); err != nil {
		return fmt.Errorf("failed to rebuild engine: %w", err)
	}

	// For each sibling, set parent to current branch
	for _, sibling := range siblings {
		if err := eng.SetParent(gctx, sibling, currentBranch); err != nil {
			return fmt.Errorf("failed to reparent %s to %s: %w", sibling, currentBranch, err)
		}
	}

	splog.Info("Folded %s into %s (kept %s).",
		tui.ColorBranchName(parent, true),
		tui.ColorBranchName(currentBranch, false),
		tui.ColorBranchName(currentBranch, false))

	// Restack current branch and all its descendants
	branchesToRestack := eng.GetRelativeStack(currentBranch, engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	if err := RestackBranches(gctx, branchesToRestack, eng, splog, ctx.RepoRoot); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}
