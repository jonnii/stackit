package fold

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

func foldWithKeep(gctx context.Context, ctx *runtime.Context, currentBranch, parentBranch engine.Branch, eng engine.Engine, splog *tui.Splog, _ Options) error {
	// Get all children of parent (siblings + current branch)
	allChildren := parentBranch.GetChildren()

	// Identify siblings (children of parent excluding current branch)
	siblings := []engine.Branch{}
	for _, child := range allChildren {
		if child.GetName() != currentBranch.GetName() {
			siblings = append(siblings, child)
		}
	}

	// Ensure we're on the current branch
	if err := eng.CheckoutBranch(gctx, currentBranch); err != nil {
		return fmt.Errorf("failed to checkout current branch: %w", err)
	}

	// Try fast-forward merge first, fallback to regular merge
	_, err := git.RunGitCommandWithContext(gctx, "merge", "--ff-only", parentBranch.GetName())
	if err != nil {
		// Fast-forward failed, try regular merge
		_, err = git.RunGitCommandWithContext(gctx, "merge", "--no-edit", parentBranch.GetName())
		if err != nil {
			return fmt.Errorf("failed to merge %s into %s due to conflicts. Please resolve the conflicts and run 'git commit', or abort with 'git merge --abort'", parentBranch.GetName(), currentBranch.GetName())
		}
	}

	// Delete the parent branch (this will reparent current branch and siblings to grandparent)
	if err := eng.DeleteBranch(gctx, parentBranch); err != nil {
		return fmt.Errorf("failed to delete parent branch: %w", err)
	}

	// Rebuild engine to reflect the deletion
	if err := eng.Rebuild(eng.Trunk().GetName()); err != nil {
		return fmt.Errorf("failed to rebuild engine: %w", err)
	}

	// For each sibling, set parent to current branch
	for _, sibling := range siblings {
		if err := eng.SetParent(gctx, sibling, currentBranch); err != nil {
			return fmt.Errorf("failed to reparent %s to %s: %w", sibling.GetName(), currentBranch.GetName(), err)
		}
	}

	splog.Info("Folded %s into %s (kept %s).",
		style.ColorBranchName(parentBranch.GetName(), true),
		style.ColorBranchName(currentBranch.GetName(), false),
		style.ColorBranchName(currentBranch.GetName(), false))

	// Restack current branch and all its descendants
	branchesToRestack := currentBranch.GetRelativeStack(engine.StackRange{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	if err := actions.RestackBranches(gctx, branchesToRestack, eng, splog, ctx.RepoRoot); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}
