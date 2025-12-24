// Package delete provides functionality for deleting branches and their metadata.
package delete

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// Options contains options for deleting branches
type Options struct {
	BranchName string
	Downstack  bool
	Force      bool
	Upstack    bool
}

// Action deletes a branch and its metadata
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branchName := opts.BranchName
	if branchName == "" {
		currentBranch := eng.CurrentBranch()
		if currentBranch == nil {
			return fmt.Errorf("no branch specified and not on a branch")
		}
		branchName = currentBranch.Name
	}

	if branchName == "" {
		return fmt.Errorf("no branch specified and not on a branch")
	}

	branch := eng.GetBranch(branchName)
	if branch.IsTrunk() {
		return fmt.Errorf("cannot delete trunk branch %s", branchName)
	}

	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Determine branches to delete
	toDelete := []string{branchName}

	if opts.Upstack {
		branch := eng.GetBranch(branchName)
		upstack := eng.GetRelativeStackUpstack(branch)
		for _, b := range upstack {
			toDelete = append(toDelete, b.Name)
		}
	}

	if opts.Downstack {
		branch := eng.GetBranch(branchName)
		downstack := eng.GetRelativeStackDownstack(branch)
		downstackNames := make([]string, len(downstack))
		for i, b := range downstack {
			downstackNames[i] = b.Name
		}
		toDelete = append(downstackNames, toDelete...)
	}

	// Confirm if not forced and not merged/closed
	if !opts.Force {
		for _, b := range toDelete {
			shouldDelete, reason := actions.ShouldDeleteBranch(ctx.Context, b, eng, false)
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
		splog.Info("Restacking children of deleted %s...", actions.Pluralize("branch", len(toDelete)))
		// Convert []string to []Branch for RestackBranches
		branches := make([]engine.Branch, len(childrenToRestack))
		for i, name := range childrenToRestack {
			branches[i] = eng.GetBranch(name)
		}
		if err := actions.RestackBranches(ctx.Context, branches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack children: %w", err)
		}
	}

	return nil
}
