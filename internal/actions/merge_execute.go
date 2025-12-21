package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// ExecuteMergePlanOptions contains options for executing a merge plan
type ExecuteMergePlanOptions struct {
	Plan  *MergePlan
	Force bool
}

// ExecuteMergePlan executes a validated merge plan step by step
func ExecuteMergePlan(ctx *runtime.Context, opts ExecuteMergePlanOptions) error {
	plan := opts.Plan
	splog := ctx.Splog

	for i, step := range plan.Steps {
		// 1. Re-validate preconditions for this step
		if err := validateStepPreconditions(step, ctx, opts); err != nil {
			return fmt.Errorf("step %d (%s) failed precondition: %w", i+1, step.Description, err)
		}

		// 2. Execute the step
		if err := executeStep(step, ctx, opts); err != nil {
			return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Description, err)
		}

		// 3. Log progress
		splog.Info("✓ %s", step.Description)
	}

	return nil
}

// validateStepPreconditions validates that a step can be executed
func validateStepPreconditions(step MergePlanStep, ctx *runtime.Context, opts ExecuteMergePlanOptions) error {
	eng := ctx.Engine
	context := ctx.Context

	switch step.StepType {
	case StepMergePR:
		// Validate PR still exists and is open
		prInfo, err := eng.GetPrInfo(context, step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}
		if prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", step.BranchName)
		}
		if prInfo.State != "OPEN" {
			return fmt.Errorf("PR #%d for branch %s is %s (not open)", *prInfo.Number, step.BranchName, prInfo.State)
		}
		// Optionally check CI checks haven't changed to failing
		if !opts.Force && ctx.GitHubClient != nil {
			passing, _, err := ctx.GitHubClient.GetPRChecksStatus(context, step.BranchName)
			if err == nil && !passing {
				return fmt.Errorf("PR #%d for branch %s has failing CI checks", *prInfo.Number, step.BranchName)
			}
		}

	case StepRestack:
		// Validate branch still exists
		if !eng.IsBranchTracked(step.BranchName) {
			return fmt.Errorf("branch %s is not tracked", step.BranchName)
		}

	case StepDeleteBranch:
		// Validate branch exists (or allow if already deleted)
		// This is non-blocking - branch might already be deleted

	case StepUpdatePRBase:
		// Validate PR exists
		prInfo, err := eng.GetPrInfo(context, step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}
		if prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", step.BranchName)
		}

	case StepPullTrunk:
		// No preconditions needed
	}

	return nil
}

// executeStep executes a single step
func executeStep(step MergePlanStep, ctx *runtime.Context, _ ExecuteMergePlanOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	context := ctx.Context

	switch step.StepType {
	case StepMergePR:
		if ctx.GitHubClient == nil {
			return fmt.Errorf("GitHub client not available")
		}
		if err := ctx.GitHubClient.MergePullRequest(context, step.BranchName); err != nil {
			return fmt.Errorf("failed to merge PR: %w", err)
		}

	case StepPullTrunk:
		pullResult, err := eng.PullTrunk(context)
		if err != nil {
			return fmt.Errorf("failed to pull trunk: %w", err)
		}
		switch pullResult {
		case engine.PullDone:
			rev, _ := eng.GetRevision(context, eng.Trunk())
			revShort := rev
			if len(rev) > 7 {
				revShort = rev[:7]
			}
			splog.Debug("Trunk fast-forwarded to %s", revShort)
		case engine.PullUnneeded:
			splog.Debug("Trunk is up to date")
		case engine.PullConflict:
			return fmt.Errorf("trunk could not be fast-forwarded (conflict)")
		}

	case StepRestack:
		// Restack the branch - RestackBranch will automatically handle reparenting
		// if the parent has been merged/deleted
		result, err := eng.RestackBranch(context, step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to restack: %w", err)
		}

		// Get the actual parent after restacking (may have been reparented)
		// Use NewParent from result if reparented, otherwise get from engine
		actualParent := result.NewParent
		if actualParent == "" {
			actualParent = eng.GetParent(step.BranchName)
		}
		if actualParent == "" {
			actualParent = eng.Trunk()
		}

		switch result.Result {
		case engine.RestackDone:
			// Success - now push the rebased branch and update PR base
			// Force push is required since we rebased
			if err := git.PushBranch(context, step.BranchName, git.GetRemote(context), true, false); err != nil {
				return fmt.Errorf("failed to push rebased branch %s: %w", step.BranchName, err)
			}
			splog.Debug("Pushed rebased branch %s to remote", step.BranchName)

			// Update the PR's base branch to the actual parent (not always trunk)
			if err := updatePRBaseBranchFromContext(ctx, step.BranchName, actualParent); err != nil {
				return fmt.Errorf("failed to update PR base for %s: %w", step.BranchName, err)
			}
			splog.Debug("Updated PR base for %s to %s", step.BranchName, actualParent)

		case engine.RestackConflict:
			// Save continuation state
			continuation := &config.ContinuationState{
				RebasedBranchBase:     result.RebasedBranchBase,
				CurrentBranchOverride: eng.CurrentBranch(),
			}
			if err := config.PersistContinuationState(ctx.RepoRoot, continuation); err != nil {
				return fmt.Errorf("failed to persist continuation: %w", err)
			}
			return fmt.Errorf("hit conflict restacking %s", step.BranchName)
		case engine.RestackUnneeded:
			// Already up to date, but still need to ensure PR base is correct
			// Push in case local is ahead of remote
			if err := git.PushBranch(context, step.BranchName, git.GetRemote(context), true, false); err != nil {
				splog.Debug("Failed to push branch %s (may already be up to date): %v", step.BranchName, err)
			}
			// Update PR base to the actual parent (not always trunk)
			if err := updatePRBaseBranchFromContext(ctx, step.BranchName, actualParent); err != nil {
				splog.Debug("Failed to update PR base for %s: %v", step.BranchName, err)
			}
		}

	case StepDeleteBranch:
		// Only delete if branch is tracked
		if eng.IsBranchTracked(step.BranchName) {
			if err := eng.DeleteBranch(context, step.BranchName); err != nil {
				// Non-fatal - branch might already be deleted
				splog.Debug("Failed to delete branch %s (may already be deleted): %v", step.BranchName, err)
			}
		}

	case StepUpdatePRBase:
		// For top-down strategy: rebase branch onto trunk and update PR base
		if err := executeUpdatePRBase(ctx, step); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown step type: %s", step.StepType)
	}

	return nil
}

// executeUpdatePRBase handles the UPDATE_PR_BASE step
// This is used in top-down strategy to rebase the current branch onto trunk
func executeUpdatePRBase(ctx *runtime.Context, step MergePlanStep) error {
	eng := ctx.Engine
	trunk := eng.Trunk()
	context := ctx.Context

	// Get the parent revision (old base)
	parent := eng.GetParent(step.BranchName)
	if parent == "" {
		parent = trunk
	}

	// Get the old parent revision
	oldParentRev, err := eng.GetRevision(context, parent)
	if err != nil {
		return fmt.Errorf("failed to get parent revision: %w", err)
	}

	// If parent is already trunk, we might just need to update the PR base
	if parent == trunk {
		// Just update the PR base branch via GitHub API
		return updatePRBaseBranchFromContext(ctx, step.BranchName, trunk)
	}

	// Rebase the branch onto trunk
	gitResult, err := git.Rebase(context, step.BranchName, trunk, oldParentRev)
	if err != nil {
		return fmt.Errorf("failed to rebase: %w", err)
	}

	if gitResult == git.RebaseConflict {
		return fmt.Errorf("rebase conflict while rebasing %s onto %s", step.BranchName, trunk)
	}

	// Update parent to trunk
	if err := eng.SetParent(context, step.BranchName, trunk); err != nil {
		return fmt.Errorf("failed to update parent: %w", err)
	}

	// Update PR base branch via GitHub API
	if err := updatePRBaseBranchFromContext(ctx, step.BranchName, trunk); err != nil {
		return fmt.Errorf("failed to update PR base: %w", err)
	}

	// Rebuild engine to reflect changes
	if err := eng.Rebuild(context, trunk); err != nil {
		return fmt.Errorf("failed to rebuild engine: %w", err)
	}

	return nil
}

// updatePRBaseBranchFromContext updates a PR's base branch via GitHub API using the runtime context's GitHubClient
func updatePRBaseBranchFromContext(ctx *runtime.Context, branchName, newBase string) error {
	if ctx.GitHubClient == nil {
		// If we can't get GitHub client, skip this step (non-fatal)
		return nil
	}

	owner, repo := ctx.GitHubClient.GetOwnerRepo()

	// Get PR for this branch
	pr, err := ctx.GitHubClient.GetPullRequestByBranch(ctx.Context, owner, repo, branchName)
	if err != nil || pr == nil {
		// PR not found or error - non-fatal
		return nil
	}

	// Update PR base
	updateOpts := github.UpdatePROptions{
		Base: &newBase,
	}

	if err := ctx.GitHubClient.UpdatePullRequest(ctx.Context, owner, repo, pr.Number, updateOpts); err != nil {
		return fmt.Errorf("failed to update PR base: %w", err)
	}

	return nil
}

// CheckSyncStatus checks if the repository is up to date with remote
func CheckSyncStatus(ctx context.Context, eng engine.Engine, splog *tui.Splog) (bool, []string, error) {
	needsSync := false
	staleBranches := []string{}

	// Check if trunk needs pulling
	pullResult, err := eng.PullTrunk(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check trunk status: %w", err)
	}

	if pullResult == engine.PullDone {
		needsSync = true
		staleBranches = append(staleBranches, eng.Trunk())
	}

	// Check all tracked branches
	allBranches := eng.AllBranchNames()
	for _, branchName := range allBranches {
		if eng.IsTrunk(branchName) {
			continue
		}

		matchesRemote, err := eng.BranchMatchesRemote(ctx, branchName)
		if err != nil {
			splog.Debug("Failed to check if %s matches remote: %v", branchName, err)
			continue
		}

		if !matchesRemote {
			needsSync = true
			staleBranches = append(staleBranches, branchName)
		}
	}

	return needsSync, staleBranches, nil
}
