package fold

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

func foldNormal(gctx context.Context, ctx *runtime.Context, currentBranch, parent string, eng engine.Engine, splog *tui.Splog, _ Options) error {
	// Checkout parent branch
	parentBranch := eng.GetBranch(parent)
	if err := git.CheckoutBranch(gctx, parentBranch.Name); err != nil {
		return fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Rebuild engine so it knows we're on the parent branch
	if err := eng.Rebuild(eng.Trunk().Name); err != nil {
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
	descendants := parentBranch.GetRelativeStack(engine.StackRange{
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
		if err := eng.Rebuild(eng.Trunk().Name); err != nil {
			return fmt.Errorf("failed to rebuild engine: %w", err)
		}

		// Get updated descendants list (current branch's children are now children of parent)
		parentBranch := eng.GetBranch(parent)
		updatedDescendants := parentBranch.GetRelativeStack(engine.StackRange{
			RecursiveChildren: true,
			IncludeCurrent:    false,
			RecursiveParents:  false,
		})

		if err := actions.RestackBranches(gctx, updatedDescendants, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}
