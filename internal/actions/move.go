package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// MoveOptions contains options for the move command
type MoveOptions struct {
	Source string // Branch to move (defaults to current branch)
	Onto   string // Branch to move onto
}

// MoveAction performs the move operation
func MoveAction(ctx *runtime.Context, opts MoveOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Default source to current branch
	source := opts.Source
	if source == "" {
		source = eng.CurrentBranch()
		if source == "" {
			return fmt.Errorf("not on a branch and no source branch specified")
		}
	}

	// Take snapshot before modifying the repository
	args := []string{}
	if opts.Source != "" {
		args = append(args, "--source", opts.Source)
	}
	if opts.Onto != "" {
		args = append(args, "--onto", opts.Onto)
	}
	if err := eng.TakeSnapshot(gctx, "move", args); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Prevent moving trunk (check before tracking check since trunk might not be tracked)
	if eng.IsTrunk(source) {
		return fmt.Errorf("cannot move trunk branch")
	}

	// Validate source exists and is tracked
	if !eng.IsBranchTracked(source) {
		return fmt.Errorf("branch %s is not tracked by Stackit", source)
	}

	// Validate onto is provided
	onto := opts.Onto
	if onto == "" {
		return fmt.Errorf("onto branch must be specified")
	}

	// Validate onto exists
	if !eng.IsTrunk(onto) && !eng.IsBranchTracked(onto) {
		// Check if it's an untracked branch
		allBranches := eng.AllBranchNames()
		found := false
		for _, b := range allBranches {
			if b == onto {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("branch %s does not exist", onto)
		}
	}

	// Prevent moving onto itself
	if source == onto {
		return fmt.Errorf("cannot move branch onto itself")
	}

	// Cycle detection: ensure onto is not a descendant of source
	descendants := eng.GetRelativeStack(source, engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})
	for _, d := range descendants {
		if d == onto {
			return fmt.Errorf("cannot move %s onto its own descendant %s", source, onto)
		}
	}

	// Get current parent for logging
	oldParent := eng.GetParent(source)
	if oldParent == "" {
		oldParent = eng.Trunk()
	}

	// Update parent in engine
	if err := eng.SetParent(gctx, source, onto); err != nil {
		return fmt.Errorf("failed to set parent: %w", err)
	}

	splog.Info("Moved %s from %s to %s.",
		tui.ColorBranchName(source, true),
		tui.ColorBranchName(oldParent, false),
		tui.ColorBranchName(onto, false))

	// Get all branches that need restacking: source and all its descendants
	branchesToRestack := eng.GetRelativeStack(source, engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	// Restack source and all its descendants
	if err := RestackBranches(gctx, branchesToRestack, eng, splog, ctx.RepoRoot); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}
