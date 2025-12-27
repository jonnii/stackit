package sync

import (
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// syncGitHubInfo synchronizes PR information from GitHub and updates local parents
func syncGitHubInfo(ctx *runtime.Context, branchesToRestack *[]string) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Sync PR info
	allBranches := eng.AllBranches()
	branchNames := make([]string, len(allBranches))
	for i, b := range allBranches {
		branchNames[i] = b.GetName()
	}

	repoOwner, repoName, _ := utils.GetRepoInfo(gctx)
	if repoOwner != "" && repoName != "" {
		if err := github.SyncPrInfo(gctx, branchNames, repoOwner, repoName, func(name string, prInfo *github.PullRequestInfo) {
			branch := eng.GetBranch(name)
			_ = eng.UpsertPrInfo(branch, engine.NewPrInfo(
				&prInfo.Number,
				prInfo.Title,
				prInfo.Body,
				prInfo.State,
				prInfo.Base,
				prInfo.HTMLURL,
				prInfo.Draft,
			))
		}); err != nil {
			// Non-fatal, continue
			splog.Debug("Failed to sync PR info: %v", err)
		}

		// Update PR body footers if needed
		if ctx.GitHubClient != nil {
			actions.UpdateStackPRMetadata(gctx, branchNames, eng, ctx.GitHubClient, repoOwner, repoName)
		}
	}

	// Synchronize local parents with GitHub PR base branches
	// This can happen even if we couldn't sync with GitHub just now,
	// using the metadata already stored in the engine.
	syncResult, err := ParentsFromGitHubBase(ctx)
	if err != nil {
		splog.Debug("Failed to sync parents from GitHub: %v", err)
	} else if len(syncResult.BranchesReparented) > 0 {
		// Add reparented branches to restack list
		for _, branchName := range syncResult.BranchesReparented {
			*branchesToRestack = append(*branchesToRestack, branchName)
			// Also add descendants
			branch := eng.GetBranch(branchName)
			upstack := eng.GetRelativeStackUpstack(branch)
			for _, b := range upstack {
				*branchesToRestack = append(*branchesToRestack, b.GetName())
			}
		}
	}

	return nil
}

// ParentsResult contains the result of synchronizing parents from GitHub
type ParentsResult struct {
	BranchesReparented []string
}

// ParentsFromGitHubBase synchronizes local parents with GitHub PR base branches
func ParentsFromGitHubBase(ctx *runtime.Context) (*ParentsResult, error) {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	allBranches := eng.AllBranches()
	reparented := []string{}

	// Map of all local branches for quick lookup
	localBranches := make(map[string]bool)
	for _, b := range allBranches {
		localBranches[b.GetName()] = true
	}
	localBranches[eng.Trunk().GetName()] = true

	for _, branch := range allBranches {
		if branch.IsTrunk() {
			continue
		}

		prInfo, err := eng.GetPrInfo(branch)
		if err != nil || prInfo == nil || prInfo.Base() == "" {
			continue
		}

		currentParent := eng.GetParent(branch)
		currentParentName := ""
		if currentParent == nil {
			currentParentName = eng.Trunk().GetName()
		} else {
			currentParentName = currentParent.GetName()
		}

		githubBase := prInfo.Base()

		// If GitHub base is different from local parent, and GitHub base is a valid local branch
		if githubBase != currentParentName && localBranches[githubBase] {
			// Before reparenting to match GitHub, check if the GitHub base is an
			// ancestor of our current local parent.
			if currentParentName != eng.Trunk().GetName() {
				isAncestor, err := eng.IsAncestor(githubBase, currentParentName)
				if err == nil && isAncestor {
					// If GitHub base is an ancestor, it's a "downgrade" in specificity.
					// We only skip reparenting if the branch is EMPTY relative to its current parent.
					// This handles the "stale PR" bug in diamond structures where 'submit'
					// skips updating the PR base because the branch is empty.
					isEmpty, err := eng.IsBranchEmpty(gctx, branch.GetName())
					if err == nil && isEmpty {
						splog.Debug("GitHub PR for %s has base %s, which is an ancestor of local parent %s. "+
							"Branch is empty relative to its parent, so keeping the more specific local parent.",
							branch.GetName(), githubBase, currentParentName)
						continue
					}
				}
			}

			splog.Info("GitHub PR for %s has base %s, but local parent is %s. Updating local parent...",
				style.ColorBranchName(branch.GetName(), false),
				style.ColorBranchName(githubBase, false),
				style.ColorBranchName(currentParentName, false))

			if err := eng.SetParent(gctx, branch, eng.GetBranch(githubBase)); err != nil {
				splog.Debug("Failed to update parent for %s: %v", branch.GetName(), err)
				continue
			}

			reparented = append(reparented, branch.GetName())
		}
	}

	return &ParentsResult{
		BranchesReparented: reparented,
	}, nil
}
