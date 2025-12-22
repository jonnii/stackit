package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// DeleteOptions contains options for deleting branches
type DeleteOptions struct {
	BranchName string
	Downstack  bool
	Force      bool
	Upstack    bool
}

// Delete deletes a branch and its metadata
func Delete(ctx *runtime.Context, opts DeleteOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branchName := opts.BranchName
	if branchName == "" {
		branchName = eng.CurrentBranch()
	}

	if branchName == "" {
		return fmt.Errorf("no branch specified and not on a branch")
	}

	if eng.IsTrunk(branchName) {
		return fmt.Errorf("cannot delete trunk branch %s", branchName)
	}

	if !eng.IsBranchTracked(branchName) {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Determine branches to delete
	toDelete := []string{branchName}

	if opts.Upstack {
		toDelete = append(toDelete, eng.GetRelativeStackUpstack(branchName)...)
	}

	if opts.Downstack {
		// Get ancestors (excluding trunk)
		curr := branchName
		for {
			parent := eng.GetParent(curr)
			if parent == "" || eng.IsTrunk(parent) {
				break
			}
			// Prepend ancestor
			toDelete = append([]string{parent}, toDelete...)
			curr = parent
		}
	}

	// Confirm if not forced and not merged/closed
	if !opts.Force {
		for _, b := range toDelete {
			shouldDelete, reason := ShouldDeleteBranch(ctx.Context, b, eng, false)
			if !shouldDelete {
				// For now, if any branch in the list shouldn't be deleted and we're not forced,
				// we might want to prompt. But since we don't have interactive prompting yet,
				// we'll just fail if it's not "safe" to delete.
				// Actually, shouldDeleteBranch returns false if it's not merged/closed/empty.

				// Let's refine this: if it's not forced, we should at least check if the branch
				// we're deleting has unmerged changes.

				// For now, if we're not forced, and shouldDeleteBranch says no, we'll ask for --force.
				if reason == "" {
					return fmt.Errorf("branch %s is not merged/closed; use --force to delete anyway", b)
				}
			}
		}
	}

	// Track children that will need restacking (only for the last branch in the stack if deleting multiple)
	// Actually, if we delete a middle branch, its children are reparented to its parent.
	// If we delete a whole stack, only children of the stack need restacking onto the stack's parent.

	// Collect all children of all branches we are deleting
	allChildren := make(map[string]bool)
	for _, b := range toDelete {
		for _, child := range eng.GetChildren(b) {
			allChildren[child] = true
		}
	}
	// Remove branches that are also being deleted from the children set
	for _, b := range toDelete {
		delete(allChildren, b)
	}

	// Delete branches
	for _, b := range toDelete {
		if err := eng.DeleteBranch(ctx.Context, b); err != nil {
			return fmt.Errorf("failed to delete branch %s: %w", b, err)
		}
		splog.Info("Deleted branch %s", tui.ColorBranchName(b, false))
	}

	// Restack children if any
	if len(allChildren) > 0 {
		childrenToRestack := []string{}
		for child := range allChildren {
			childrenToRestack = append(childrenToRestack, child)
		}

		// Sort children to maintain stack order if possible?
		// Actually, RestackBranches handles multiple branches.

		splog.Info("Restacking children of deleted %s...", Pluralize("branch", len(toDelete)))
		if err := RestackBranches(ctx.Context, childrenToRestack, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack children: %w", err)
		}
	}

	return nil
}
