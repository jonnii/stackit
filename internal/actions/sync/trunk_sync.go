package sync

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
)

// syncTrunk handles pulling the trunk and resolving any conflicts
func syncTrunk(ctx *runtime.Context, opts *Options) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context
	trunk := eng.Trunk()
	trunkName := trunk.Name

	// Pull trunk
	splog.Info("Pulling %s from remote...", style.ColorBranchName(trunkName, false))
	pullResult, err := eng.PullTrunk(gctx)
	if err != nil {
		return fmt.Errorf("failed to pull trunk: %w", err)
	}

	switch pullResult {
	case engine.PullDone:
		trunk := eng.Trunk()
		rev, _ := trunk.GetRevision()
		revShort := rev
		if len(rev) > 7 {
			revShort = rev[:7]
		}
		splog.Info("%s fast-forwarded to %s.",
			style.ColorBranchName(trunkName, true),
			style.ColorDim(revShort))
	case engine.PullUnneeded:
		splog.Info("%s is up to date.", style.ColorBranchName(trunkName, true))
	case engine.PullConflict:
		splog.Warn("%s could not be fast-forwarded.", style.ColorBranchName(trunkName, false))

		// Prompt to overwrite (or use force flag)
		shouldReset := opts.Force
		if !shouldReset {
			// For now, if not force and interactive, we'll skip
			// In a full implementation, we would prompt here
			splog.Info("Skipping trunk reset. Use --force to overwrite trunk with remote version.")
		}

		if shouldReset {
			if err := eng.ResetTrunkToRemote(gctx); err != nil {
				return fmt.Errorf("failed to reset trunk: %w", err)
			}
			trunk := eng.Trunk()
			rev, _ := trunk.GetRevision()
			revShort := rev
			if len(rev) > 7 {
				revShort = rev[:7]
			}
			splog.Info("%s set to %s.",
				style.ColorBranchName(trunkName, true),
				style.ColorDim(revShort))
		}
	}

	return nil
}
