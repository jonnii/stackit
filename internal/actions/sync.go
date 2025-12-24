package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
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

	// Clean branches (delete merged/closed)
	branchesToRestack := []string{}

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

		// Synchronize local parents with GitHub PR base branches
		syncResult, err := SyncParentsFromGitHubBase(ctx)
		if err != nil {
			splog.Debug("Failed to sync parents from GitHub: %v", err)
		} else if len(syncResult.BranchesReparented) > 0 {
			// Add reparented branches to restack list
			for _, branchName := range syncResult.BranchesReparented {
				branchesToRestack = append(branchesToRestack, branchName)
				// Also add descendants
				branch := eng.GetBranch(branchName)
				upstack := eng.GetRelativeStackUpstack(branch)
				for _, b := range upstack {
					branchesToRestack = append(branchesToRestack, b.Name)
				}
			}
		}
	}

	// Clean branches (delete merged/closed)
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

// SyncParentsResult contains the result of synchronizing parents from GitHub
type SyncParentsResult struct {
	BranchesReparented []string
}

// SyncParentsFromGitHubBase synchronizes local parents with GitHub PR base branches
func SyncParentsFromGitHubBase(ctx *runtime.Context) (*SyncParentsResult, error) {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	allBranches := eng.AllBranches()
	reparented := []string{}

	// Map of all local branches for quick lookup
	localBranches := make(map[string]bool)
	for _, b := range allBranches {
		localBranches[b.Name] = true
	}
	localBranches[eng.Trunk().Name] = true

	for _, branch := range allBranches {
		if branch.IsTrunk() {
			continue
		}

		prInfo, err := eng.GetPrInfo(branch.Name)
		if err != nil || prInfo == nil || prInfo.Base == "" {
			continue
		}

		currentParent := eng.GetParent(branch)
		currentParentName := ""
		if currentParent == nil {
			currentParentName = eng.Trunk().Name
		} else {
			currentParentName = currentParent.Name
		}

		githubBase := prInfo.Base

		// If GitHub base is different from local parent, and GitHub base is a valid local branch
		if githubBase != currentParentName && localBranches[githubBase] {
			// CRITICAL: Before reparenting to match GitHub, check if the GitHub base is an
			// ancestor of our current local parent. If it is, then our local parent is
			// "more specific" (closer to the branch in the stack) than GitHub's base.
			// This often happens in diamond structures if a PR base update failed or is stale.
			if currentParentName != eng.Trunk().Name {
				isAncestor, err := git.IsAncestor(githubBase, currentParentName)
				if err == nil && isAncestor {
					splog.Debug("GitHub PR for %s has base %s, which is an ancestor of local parent %s. Keeping the more specific local parent.",
						branch.Name, githubBase, currentParentName)
					continue
				}
			}

			splog.Info("GitHub PR for %s has base %s, but local parent is %s. Updating local parent...",
				tui.ColorBranchName(branch.Name, false),
				tui.ColorBranchName(githubBase, false),
				tui.ColorBranchName(currentParentName, false))

			if err := eng.SetParent(gctx, branch.Name, githubBase); err != nil {
				splog.Debug("Failed to update parent for %s: %v", branch.Name, err)
				continue
			}

			reparented = append(reparented, branch.Name)
		}
	}

	return &SyncParentsResult{
		BranchesReparented: reparented,
	}, nil
}
