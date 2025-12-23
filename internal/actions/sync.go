package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// SyncOptions contains options for the sync command
type SyncOptions struct {
	All     bool
	Force   bool
	Restack bool
}

// SyncAction performs the sync operation
func SyncAction(ctx *runtime.Context, opts SyncOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context
	trunk := eng.Trunk() // Cache trunk for this function scope

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
	splog.Info("Pulling %s from remote...", tui.ColorBranchName(trunk, false))
	pullResult, err := eng.PullTrunk(gctx)
	if err != nil {
		return fmt.Errorf("failed to pull trunk: %w", err)
	}

	switch pullResult {
	case engine.PullDone:
		rev, _ := eng.GetRevision(trunk)
		revShort := rev
		if len(rev) > 7 {
			revShort = rev[:7]
		}
		splog.Info("%s fast-forwarded to %s.",
			tui.ColorBranchName(trunk, true),
			tui.ColorDim(revShort))
	case engine.PullUnneeded:
		splog.Info("%s is up to date.", tui.ColorBranchName(trunk, true))
	case engine.PullConflict:
		splog.Warn("%s could not be fast-forwarded.", tui.ColorBranchName(trunk, false))

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
			rev, _ := eng.GetRevision(trunk)
			revShort := rev
			if len(rev) > 7 {
				revShort = rev[:7]
			}
			splog.Info("%s set to %s.",
				tui.ColorBranchName(trunk, true),
				tui.ColorDim(revShort))
		}
	}

	// Sync PR info
	allBranches := eng.AllBranchNames()
	repoOwner, repoName, _ := utils.GetRepoInfo(gctx)
	if repoOwner != "" && repoName != "" {
		if err := github.SyncPrInfo(gctx, allBranches, repoOwner, repoName); err != nil {
			// Non-fatal, continue
			splog.Debug("Failed to sync PR info: %v", err)
		}
	}

	// Clean branches (delete merged/closed)
	branchesToRestack := []string{}

	cleanResult, err := CleanBranches(ctx, CleanBranchesOptions{
		Force: opts.Force,
	})
	if err != nil {
		return fmt.Errorf("failed to clean branches: %w", err)
	}

	// Add branches with new parents to restack list
	for _, branchName := range cleanResult.BranchesWithNewParents {
		upstack := eng.GetRelativeStackUpstack(branchName)
		branchesToRestack = append(branchesToRestack, upstack...)
		branchesToRestack = append(branchesToRestack, branchName)
	}

	// Restack if requested
	if !opts.Restack {
		splog.Tip("Try the --restack flag to automatically restack the current stack.")
		return nil
	}

	// Add current branch stack to restack list
	currentBranch := eng.CurrentBranch()
	if currentBranch != "" {
		currentBranchObj := eng.GetBranch(currentBranch)
		if currentBranchObj.IsTracked() {
			// Get full stack (up to trunk)
			stack := eng.GetFullStack(currentBranch)
			// Add branches to restack list
			branchesToRestack = append(branchesToRestack, stack...)
		} else if currentBranchObj.IsTrunk() {
			// If on trunk, restack all branches
			stack := eng.GetRelativeStack(currentBranch, engine.Scope{RecursiveChildren: true})
			branchesToRestack = append(branchesToRestack, stack...)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueBranches := []string{}
	for _, branchName := range branchesToRestack {
		if !seen[branchName] {
			seen[branchName] = true
			uniqueBranches = append(uniqueBranches, branchName)
		}
	}

	// Sort branches topologically (parents before children) for correct restack order
	sortedBranches := eng.SortBranchesTopologically(uniqueBranches)

	// Restack branches
	if len(sortedBranches) > 0 {
		if err := RestackBranches(gctx, sortedBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}
