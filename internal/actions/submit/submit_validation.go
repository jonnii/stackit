// Package submit provides functionality for submitting stacked branches as pull requests.
package submit

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// ValidateBranchesToSubmit validates that branches are ready to submit
func ValidateBranchesToSubmit(ctx context.Context, branches []string, eng engine.Engine, runtimeCtx *runtime.Context) error {
	// Sync PR info first
	repoOwner, repoName, _ := utils.GetRepoInfo(ctx)
	if repoOwner != "" && repoName != "" {
		if err := github.SyncPrInfo(ctx, branches, repoOwner, repoName); err != nil {
			// Non-fatal, continue
			runtimeCtx.Splog.Debug("Failed to sync PR info: %v", err)
		}
	}

	// Validate base revisions
	if err := validateBaseRevisions(ctx, branches, eng, runtimeCtx); err != nil {
		return err
	}

	// Validate no empty branches
	if err := validateNoEmptyBranches(ctx, branches, eng, runtimeCtx); err != nil {
		return err
	}

	// Validate no merged/closed branches
	if err := validateNoMergedOrClosedBranches(ctx, branches, eng, runtimeCtx); err != nil {
		return err
	}

	return nil
}

// validateBaseRevisions ensures that for each branch:
// 1. Its parent is trunk, OR
// 2. We are submitting its parent before it and it does not need restacking, OR
// 3. Its base matches the existing head for its parent's PR
func validateBaseRevisions(ctx context.Context, branches []string, eng engine.Engine, runtimeCtx *runtime.Context) error {
	validatedBranches := make(map[string]bool)

	for _, branchName := range branches {
		parentBranchName := eng.GetParentPrecondition(branchName)

		switch {
		case eng.IsTrunk(parentBranchName):
			if !eng.IsBranchFixed(ctx, branchName) {
				runtimeCtx.Splog.Info("Note that %s has fallen behind trunk. You may encounter conflicts if you attempt to merge it.",
					tui.ColorBranchName(branchName, false))
			}
		case validatedBranches[parentBranchName]:
			// Parent is in the submission list
			if !eng.IsBranchFixed(ctx, branchName) {
				return fmt.Errorf("you are trying to submit at least one branch that has not been restacked on its parent. To resolve this, check out %s and run 'stackit restack'",
					tui.ColorBranchName(branchName, false))
			}
		default:
			// Parent is not in submission list
			matchesRemote, err := eng.BranchMatchesRemote(ctx, parentBranchName)
			if err != nil {
				return fmt.Errorf("failed to check if parent branch matches remote: %w", err)
			}
			if !matchesRemote {
				return fmt.Errorf("you are trying to submit at least one branch whose base does not match its parent remotely, without including its parent. You may want to use 'stackit submit --stack' to ensure that the ancestors of %s are included in your submission",
					tui.ColorBranchName(branchName, false))
			}
		}

		validatedBranches[branchName] = true
	}

	return nil
}

// validateNoEmptyBranches checks for empty branches and prompts user if found
func validateNoEmptyBranches(ctx context.Context, branches []string, eng engine.BranchReader, runtimeCtx *runtime.Context) error {
	emptyBranches := []string{}
	for _, branchName := range branches {
		isEmpty, err := eng.IsBranchEmpty(ctx, branchName)
		if err != nil {
			continue
		}
		if isEmpty {
			emptyBranches = append(emptyBranches, branchName)
		}
	}

	if len(emptyBranches) == 0 {
		return nil
	}

	hasMultiple := len(emptyBranches) > 1
	runtimeCtx.Splog.Warn("The following branch%s have no changes:", actions.PluralSuffix(hasMultiple))
	for _, b := range emptyBranches {
		runtimeCtx.Splog.Warn("▸ %s", b)
	}
	runtimeCtx.Splog.Warn("Are you sure you want to submit %s?", actions.PluralIt(hasMultiple))

	// For now, we'll allow empty branches (non-interactive mode)
	// In interactive mode, we would prompt here
	// TODO: Add interactive prompt when needed

	return nil
}

// validateNoMergedOrClosedBranches checks for merged/closed PRs and prompts user if found
func validateNoMergedOrClosedBranches(ctx context.Context, branches []string, eng engine.Engine, runtimeCtx *runtime.Context) error {
	mergedOrClosedBranches := []string{}
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(ctx, branchName)
		if err != nil {
			continue
		}
		if prInfo != nil && (prInfo.State == "MERGED" || prInfo.State == "CLOSED") {
			mergedOrClosedBranches = append(mergedOrClosedBranches, branchName)
		}
	}

	if len(mergedOrClosedBranches) == 0 {
		return nil
	}

	hasMultiple := len(mergedOrClosedBranches) > 1
	runtimeCtx.Splog.Tip("You can use 'stackit sync' to find and delete all merged/closed branches automatically and rebase their children.")
	runtimeCtx.Splog.Warn("PR%s for the following branch%s already been merged or closed:", actions.PluralSuffix(hasMultiple), actions.PluralSuffix(hasMultiple))
	for _, b := range mergedOrClosedBranches {
		runtimeCtx.Splog.Warn("▸ %s", b)
	}

	// For now, we'll clear PR info and allow creating new PRs (non-interactive mode)
	// In interactive mode, we would prompt here
	// TODO: Add interactive prompt when needed
	for _, branchName := range mergedOrClosedBranches {
		// Clear PR info to allow creating new PR
		_ = eng.UpsertPrInfo(ctx, branchName, &engine.PrInfo{})
	}

	return nil
}
