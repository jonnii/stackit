package sync

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the sync command
type Options struct {
	All     bool
	Force   bool
	Restack bool
}

// Action performs the sync operation
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Handle --all flag (stub for now)
	if opts.All {
		// For now, just sync the current trunk
		// In the future, this would sync across all configured trunks
		splog.Info("Syncing branches across all configured trunks...")
	}

	// Check for uncommitted changes
	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them before syncing")
	}

	// Pull trunk
	if err := syncTrunk(ctx, &opts); err != nil {
		return err
	}

	// Clean branches (delete merged/closed)
	branchesToRestack := []string{}

	// Sync PR info
	if err := syncGitHubInfo(ctx, &branchesToRestack); err != nil {
		return err
	}

	// Clean branches (delete merged/closed)
	cleanResult, err := cleanBranches(ctx, &opts)
	if err != nil {
		return fmt.Errorf("failed to clean branches: %w", err)
	}

	// Add branches with new parents to restack list
	for _, branchName := range cleanResult.BranchesWithNewParents {
		branch := eng.GetBranch(branchName)
		upstack := eng.GetRelativeStackUpstack(branch)
		for _, b := range upstack {
			branchesToRestack = append(branchesToRestack, b.Name)
		}
		branchesToRestack = append(branchesToRestack, branchName)
	}

	// Restack if requested
	if !opts.Restack {
		splog.Tip("Try the --restack flag to automatically restack the current stack.")
		return nil
	}

	return restackBranches(ctx, branchesToRestack)
}
