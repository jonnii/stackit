package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// SquashOptions contains options for the squash command
type SquashOptions struct {
	Message string
	NoEdit  bool
}

// SquashAction performs the squash operation
func SquashAction(ctx *runtime.Context, opts SquashOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	context := ctx.Context

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch.Name == "" {
		return fmt.Errorf("not on a branch")
	}

	// Take snapshot before modifying the repository
	args := []string{}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	}
	if err := eng.TakeSnapshot("squash", args); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Squash current branch
	if err := eng.SquashCurrentBranch(context, engine.SquashOptions{
		Message: opts.Message,
		NoEdit:  opts.NoEdit,
	}); err != nil {
		return fmt.Errorf("failed to squash branch: %w", err)
	}

	splog.Info("Squashed commits in %s.", tui.ColorBranchName(currentBranch.Name, true))

	// Get upstack branches (recursive children only, excluding current branch)
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := eng.GetRelativeStack(currentBranch.Name, scope)

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := RestackBranches(context, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
