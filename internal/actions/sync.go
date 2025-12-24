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
	trunkName := trunk.Name

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
	splog.Info("Pulling %s from remote...", tui.ColorBranchName(trunkName, false))
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
			tui.ColorBranchName(trunkName, true),
			tui.ColorDim(revShort))
	case engine.PullUnneeded:
		splog.Info("%s is up to date.", tui.ColorBranchName(trunkName, true))
	case engine.PullConflict:
		splog.Warn("%s could not be fast-forwarded.", tui.ColorBranchName(trunkName, false))

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
				tui.ColorBranchName(trunkName, true),
				tui.ColorDim(revShort))
		}
	}

	// Sync PR info
	allBranches := eng.AllBranches()
	branchNames := make([]string, len(allBranches))
	for i, b := range allBranches {
		branchNames[i] = b.Name
	}
	repoOwner, repoName, _ := utils.GetRepoInfo(gctx)
	if repoOwner != "" && repoName != "" {
		if err := github.SyncPrInfo(gctx, branchNames, repoOwner, repoName); err != nil {
			// Non-fatal, continue
			splog.Debug("Failed to sync PR info: %v", err)
		}

		// Update PR body footers if needed
		if ctx.GitHubClient != nil {
			UpdateStackPRMetadata(gctx, branchNames, eng, ctx.GitHubClient, repoOwner, repoName)
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

	// Add current branch stack to restack list
	currentBranch := eng.CurrentBranch()
	if currentBranch != nil {
		if currentBranch.IsTracked() {
			// Get full stack (up to trunk)
			stack := eng.GetFullStack(*currentBranch)
			// Add branches to restack list
			for _, b := range stack {
				branchesToRestack = append(branchesToRestack, b.Name)
			}
		} else if currentBranch.IsTrunk() {
			// If on trunk, restack all branches
			stack := currentBranch.GetRelativeStack(engine.StackRange{RecursiveChildren: true})
			for _, b := range stack {
				branchesToRestack = append(branchesToRestack, b.Name)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueBranches := []engine.Branch{}
	for _, branchName := range branchesToRestack {
		if !seen[branchName] {
			seen[branchName] = true
			uniqueBranches = append(uniqueBranches, eng.GetBranch(branchName))
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
