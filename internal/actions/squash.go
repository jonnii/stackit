package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
)

// SquashOptions are options for the squash command
type SquashOptions struct {
	Message  string
	NoEdit   bool
	Engine   engine.Engine
	Splog    *tui.Splog
	RepoRoot string
}

// SquashAction performs the squash operation
func SquashAction(opts SquashOptions) error {
	// Get current branch
	currentBranch := opts.Engine.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Squash current branch
	if err := opts.Engine.SquashCurrentBranch(engine.SquashOptions{
		Message: opts.Message,
		NoEdit:  opts.NoEdit,
	}); err != nil {
		return fmt.Errorf("failed to squash branch: %w", err)
	}

	opts.Splog.Info("Squashed commits in %s.", tui.ColorBranchName(currentBranch, true))

	// Get upstack branches (recursive children only, excluding current branch)
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := opts.Engine.GetRelativeStack(currentBranch, scope)

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := RestackBranches(upstackBranches, opts.Engine, opts.Splog, opts.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
