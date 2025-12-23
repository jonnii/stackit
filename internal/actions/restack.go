package actions

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// RestackOptions contains options for the restack command
type RestackOptions struct {
	BranchName string
	Scope      engine.Scope
}

// RestackAction performs the restack operation
func RestackAction(ctx *runtime.Context, opts RestackOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get branches to restack based on scope
	branch := eng.GetBranch(opts.BranchName)
	branches := branch.GetRelativeStack(opts.Scope)

	if len(branches) == 0 {
		splog.Info("No branches to restack.")
		return nil
	}

	// Take snapshot before modifying the repository
	args := []string{}
	if opts.BranchName != "" {
		args = append(args, opts.BranchName)
	}
	if err := eng.TakeSnapshot("restack", args); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Call RestackBranches (from common.go)
	return RestackBranches(ctx.Context, branches, eng, splog, ctx.RepoRoot)
}
