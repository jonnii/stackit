package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// ValidateBranchesToSubmit validates that branches are ready to submit
func ValidateBranchesToSubmit(branches []string, eng engine.Engine, ctx *runtime.Context) error {
	// Sync PR info first
	repoOwner, repoName, _ := getRepoInfo()
	if repoOwner != "" && repoName != "" {
		if err := git.SyncPrInfo(branches, repoOwner, repoName); err != nil {
			// Non-fatal, continue
			ctx.Splog.Debug("Failed to sync PR info: %v", err)
		}
	}

	// Validate base revisions
	if err := validateBaseRevisions(branches, eng, ctx); err != nil {
		return err
	}

	// Validate no empty branches
	if err := validateNoEmptyBranches(branches, eng, ctx); err != nil {
		return err
	}

	// Validate no merged/closed branches
	if err := validateNoMergedOrClosedBranches(branches, eng, ctx); err != nil {
		return err
	}

	return nil
}

// validateBaseRevisions ensures that for each branch:
// 1. Its parent is trunk, OR
// 2. We are submitting its parent before it and it does not need restacking, OR
// 3. Its base matches the existing head for its parent's PR
func validateBaseRevisions(branches []string, eng engine.Engine, ctx *runtime.Context) error {
	validatedBranches := make(map[string]bool)

	for _, branchName := range branches {
		parentBranchName := eng.GetParentPrecondition(branchName)

		if eng.IsTrunk(parentBranchName) {
			if !eng.IsBranchFixed(branchName) {
				ctx.Splog.Info("Note that %s has fallen behind trunk. You may encounter conflicts if you attempt to merge it.",
					output.ColorBranchName(branchName, false))
			}
		} else if validatedBranches[parentBranchName] {
			// Parent is in the submission list
			if !eng.IsBranchFixed(branchName) {
				return fmt.Errorf("you are trying to submit at least one branch that has not been restacked on its parent. To resolve this, check out %s and run 'stackit restack'",
					output.ColorBranchName(branchName, false))
			}
		} else {
			// Parent is not in submission list
			matchesRemote, err := eng.BranchMatchesRemote(parentBranchName)
			if err != nil {
				return fmt.Errorf("failed to check if parent branch matches remote: %w", err)
			}
			if !matchesRemote {
				return fmt.Errorf("you are trying to submit at least one branch whose base does not match its parent remotely, without including its parent. You may want to use 'stackit submit --stack' to ensure that the ancestors of %s are included in your submission",
					output.ColorBranchName(branchName, false))
			}
		}

		validatedBranches[branchName] = true
	}

	return nil
}

// validateNoEmptyBranches checks for empty branches and prompts user if found
func validateNoEmptyBranches(branches []string, eng engine.Engine, ctx *runtime.Context) error {
	emptyBranches := []string{}
	for _, branchName := range branches {
		isEmpty, err := eng.IsBranchEmpty(branchName)
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
	ctx.Splog.Warn("The following branch%s have no changes:", pluralSuffix(hasMultiple))
	for _, b := range emptyBranches {
		ctx.Splog.Warn("▸ %s", b)
	}
	ctx.Splog.Warn("Are you sure you want to submit %s?", pluralIt(hasMultiple))

	// For now, we'll allow empty branches (non-interactive mode)
	// In interactive mode, we would prompt here
	// TODO: Add interactive prompt when needed

	return nil
}

// validateNoMergedOrClosedBranches checks for merged/closed PRs and prompts user if found
func validateNoMergedOrClosedBranches(branches []string, eng engine.Engine, ctx *runtime.Context) error {
	mergedOrClosedBranches := []string{}
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(branchName)
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
	ctx.Splog.Tip("You can use 'stackit sync' to find and delete all merged/closed branches automatically and rebase their children.")
	ctx.Splog.Warn("PR%s for the following branch%s already been merged or closed:", pluralSuffix(hasMultiple), pluralSuffix(hasMultiple))
	for _, b := range mergedOrClosedBranches {
		ctx.Splog.Warn("▸ %s", b)
	}

	// For now, we'll clear PR info and allow creating new PRs (non-interactive mode)
	// In interactive mode, we would prompt here
	// TODO: Add interactive prompt when needed
	for _, branchName := range mergedOrClosedBranches {
		// Clear PR info to allow creating new PR
		eng.UpsertPrInfo(branchName, &engine.PrInfo{})
	}

	return nil
}

// Helper functions for pluralization
func pluralSuffix(plural bool) string {
	if plural {
		return "es"
	}
	return ""
}

func pluralIt(plural bool) string {
	if plural {
		return "them"
	}
	return "it"
}
