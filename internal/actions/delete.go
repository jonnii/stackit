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
		toDelete = append(eng.GetRelativeStackDownstack(branchName), toDelete...)
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

	// Delete branches and get children to restack
	childrenToRestack, err := eng.DeleteBranches(ctx.Context, toDelete)
	if err != nil {
		return err
	}

	for _, b := range toDelete {
		splog.Info("Deleted branch %s", tui.ColorBranchName(b, false))
	}

	// Restack children if any
	if len(childrenToRestack) > 0 {
		splog.Info("Restacking children of deleted %s...", Pluralize("branch", len(toDelete)))
		if err := RestackBranches(ctx.Context, childrenToRestack, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack children: %w", err)
		}
	}

	return nil
}
