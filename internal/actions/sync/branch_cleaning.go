package sync

import (
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// cleanBranches handles cleaning merged/closed branches
func cleanBranches(ctx *runtime.Context, opts *Options) (*actions.CleanBranchesResult, error) {
	return actions.CleanBranches(ctx, actions.CleanBranchesOptions{
		Force: opts.Force,
	})
}
