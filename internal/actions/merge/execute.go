package merge

import (
	"context"
	"fmt"
	"time"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

const (
	prStateOpen = "OPEN"
)

// ProgressReporter is an interface for reporting merge progress
type ProgressReporter interface {
	StepStarted(stepIndex int, description string)
	StepCompleted(stepIndex int)
	StepFailed(stepIndex int, err error)
	StepWaiting(stepIndex int, elapsed, timeout time.Duration)
}

// mergeExecuteEngine is a minimal interface needed for executing a merge plan
type mergeExecuteEngine interface {
	engine.PRManager
	engine.BranchReader
	engine.BranchWriter
	engine.SyncManager
}

// ExecuteOptions contains options for executing a merge plan
type ExecuteOptions struct {
	Plan     *Plan
	Force    bool
	Reporter ProgressReporter // Optional progress reporter
}

// Execute executes a validated merge plan step by step
func Execute(ctx context.Context, eng mergeExecuteEngine, splog *tui.Splog, githubClient github.Client, repoRoot string, opts ExecuteOptions) error {
	plan := opts.Plan

	// If no reporter provided and we're in a TTY, use TUI
	if opts.Reporter == nil && tui.IsTTY() {
		reporter := tui.NewChannelMergeProgressReporter()

		// Calculate groups for the TUI
		groups := calculateGroups(plan)

		// Extract step descriptions
		stepDescriptions := make([]string, len(plan.Steps))
		for i, step := range plan.Steps {
			stepDescriptions[i] = step.Description
		}

		// Start TUI in a goroutine
		done := make(chan bool, 1)
		tuiErr := make(chan error, 1)
		go func() {
			err := tui.RunMergeTUI(groups, stepDescriptions, reporter.Updates(), done)
			if err != nil {
				tuiErr <- err
			}
		}()

		// Update opts to use the reporter
		opts.Reporter = reporter

		// Execute plan (this will send updates to the reporter)
		err := executeSteps(ctx, eng, splog, githubClient, repoRoot, opts)

		// Close reporter to signal TUI to finish
		reporter.Close()

		// Wait for TUI to finish or error
		select {
		case <-done:
			// TUI finished normally
		case err := <-tuiErr:
			if err != nil {
				splog.Debug("TUI error: %v", err)
			}
		}

		return err
	}

	// Execute without TUI
	return executeSteps(ctx, eng, splog, githubClient, repoRoot, opts)
}

// ExecuteInWorktree executes the merge plan in a temporary worktree
func ExecuteInWorktree(_ context.Context, _ mergeExecuteEngine, _ *tui.Splog, _ github.Client, _ string, _ ExecuteOptions) error {
	return fmt.Errorf("ExecuteInWorktree not implemented yet")
}

func calculateGroups(plan *Plan) []tui.MergeGroup {
	var groups []tui.MergeGroup
	assigned := make(map[int]bool)

	// 1. Create groups for each branch being merged
	for _, branchInfo := range plan.BranchesToMerge {
		var indices []int
		for i, step := range plan.Steps {
			if step.BranchName == branchInfo.BranchName {
				indices = append(indices, i)
				assigned[i] = true
			}
		}
		if len(indices) > 0 {
			groups = append(groups, tui.MergeGroup{
				Label:       fmt.Sprintf("PR #%d (%s)", branchInfo.PRNumber, branchInfo.BranchName),
				StepIndices: indices,
			})
		}
	}

	// 2. Create group for upstack branches
	if len(plan.UpstackBranches) > 0 {
		var indices []int
		for i, step := range plan.Steps {
			if assigned[i] {
				continue
			}
			for _, ub := range plan.UpstackBranches {
				if step.BranchName == ub {
					indices = append(indices, i)
					assigned[i] = true
					break
				}
			}
		}
		if len(indices) > 0 {
			groups = append(groups, tui.MergeGroup{
				Label:       "Restack upstack branches",
				StepIndices: indices,
			})
		}
	}

	// 3. Remaining steps (like PullTrunk)
	for i := 0; i < len(plan.Steps); i++ {
		if assigned[i] {
			continue
		}

		label := plan.Steps[i].Description
		if plan.Steps[i].StepType == StepPullTrunk {
			label = "Sync trunk"
		}

		groups = append(groups, tui.MergeGroup{
			Label:       label,
			StepIndices: []int{i},
		})
		assigned[i] = true
	}

	return groups
}

// executeSteps executes the merge plan steps
func executeSteps(ctx context.Context, eng mergeExecuteEngine, splog *tui.Splog, githubClient github.Client, repoRoot string, opts ExecuteOptions) error {
	plan := opts.Plan

	for i, step := range plan.Steps {
		// Report step started
		if opts.Reporter != nil {
			opts.Reporter.StepStarted(i, step.Description)
		}

		// 1. Re-validate preconditions for this step
		if err := validateStepPreconditions(ctx, step, eng, githubClient, opts); err != nil {
			if opts.Reporter != nil {
				opts.Reporter.StepFailed(i, err)
			}
			return fmt.Errorf("step %d (%s) failed precondition: %w", i+1, step.Description, err)
		}

		// 2. Execute the step (with progress reporting for wait steps)
		if err := executeStepWithProgress(ctx, step, i, eng, splog, githubClient, repoRoot, opts); err != nil {
			if opts.Reporter != nil {
				opts.Reporter.StepFailed(i, err)
			}
			return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Description, err)
		}

		// 3. Report step completed
		if opts.Reporter != nil {
			opts.Reporter.StepCompleted(i)
		}

		// 4. Log progress (if no reporter, use simple logging)
		if opts.Reporter == nil {
			splog.Info("✓ %s", step.Description)
		}
	}

	return nil
}

// validateStepPreconditions validates that a step can be executed
func validateStepPreconditions(ctx context.Context, step PlanStep, eng mergeExecuteEngine, githubClient github.Client, opts ExecuteOptions) error {
	switch step.StepType {
	case StepMergePR:
		// Validate PR still exists and is open
		prInfo, err := eng.GetPrInfo(step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}
		if prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", step.BranchName)
		}
		if prInfo.State != prStateOpen {
			return fmt.Errorf("PR #%d for branch %s is %s (not open)", *prInfo.Number, step.BranchName, prInfo.State)
		}
		// Optionally check CI checks haven't changed to failing or pending
		if !opts.Force && githubClient != nil {
			passing, pending, err := githubClient.GetPRChecksStatus(ctx, step.BranchName)
			if err == nil {
				if !passing {
					return fmt.Errorf("PR #%d for branch %s has failing CI checks", *prInfo.Number, step.BranchName)
				}
				if pending {
					return fmt.Errorf("PR #%d for branch %s has pending CI checks", *prInfo.Number, step.BranchName)
				}
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
		prInfo, err := eng.GetPrInfo(step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}
		if prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", step.BranchName)
		}

	case StepPullTrunk:
		// No preconditions needed

	case StepWaitCI:
		// Validate PR exists and is open
		prInfo, err := eng.GetPrInfo(step.BranchName)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}
		if prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", step.BranchName)
		}
		if prInfo.State != prStateOpen {
			return fmt.Errorf("PR #%d for branch %s is %s (not open)", *prInfo.Number, step.BranchName, prInfo.State)
		}
	}

	return nil
}

// executeStepWithProgress executes a step with progress reporting
func executeStepWithProgress(ctx context.Context, step PlanStep, stepIndex int, eng mergeExecuteEngine, splog *tui.Splog, githubClient github.Client, repoRoot string, opts ExecuteOptions) error {
	// Special handling for wait steps to report progress
	if step.StepType == StepWaitCI {
		return executeWaitCIWithProgress(ctx, step, stepIndex, eng, splog, githubClient, opts)
	}
	return executeStep(ctx, step, eng, splog, githubClient, repoRoot, opts)
}

// executeStep executes a single step
func executeStep(ctx context.Context, step PlanStep, eng mergeExecuteEngine, splog *tui.Splog, githubClient github.Client, repoRoot string, _ ExecuteOptions) error {
	switch step.StepType {
	case StepMergePR:
		if githubClient == nil {
			return fmt.Errorf("GitHub client not available")
		}
		if err := githubClient.MergePullRequest(ctx, step.BranchName); err != nil {
			return fmt.Errorf("failed to merge PR: %w", err)
		}

	case StepPullTrunk:
		pullResult, err := eng.PullTrunk(ctx)
		if err != nil {
			return fmt.Errorf("failed to pull trunk: %w", err)
		}
		switch pullResult {
		case engine.PullDone:
			rev, _ := eng.GetRevision(ctx, eng.Trunk())
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
		result, err := eng.RestackBranch(ctx, step.BranchName)
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
			if err := git.PushBranch(ctx, step.BranchName, git.GetRemote(), true, false); err != nil {
				return fmt.Errorf("failed to push rebased branch %s: %w", step.BranchName, err)
			}
			splog.Debug("Pushed rebased branch %s to remote", step.BranchName)

			// Update the PR's base branch to the actual parent (not always trunk)
			if err := updatePRBaseBranchFromContext(ctx, githubClient, step.BranchName, actualParent); err != nil {
				return fmt.Errorf("failed to update PR base for %s: %w", step.BranchName, err)
			}
			splog.Debug("Updated PR base for %s to %s", step.BranchName, actualParent)

		case engine.RestackConflict:
			// Save continuation state
			continuation := &config.ContinuationState{
				RebasedBranchBase:     result.RebasedBranchBase,
				CurrentBranchOverride: eng.CurrentBranch(),
			}
			if err := config.PersistContinuationState(repoRoot, continuation); err != nil {
				return fmt.Errorf("failed to persist continuation: %w", err)
			}
			return fmt.Errorf("hit conflict restacking %s", step.BranchName)
		case engine.RestackUnneeded:
			// Already up to date, but still need to ensure PR base is correct
			// Push in case local is ahead of remote
			if err := git.PushBranch(ctx, step.BranchName, git.GetRemote(), true, false); err != nil {
				splog.Debug("Failed to push branch %s (may already be up to date): %v", step.BranchName, err)
			}
			// Update PR base to the actual parent (not always trunk)
			if err := updatePRBaseBranchFromContext(ctx, githubClient, step.BranchName, actualParent); err != nil {
				splog.Debug("Failed to update PR base for %s: %v", step.BranchName, err)
			}
		}

	case StepDeleteBranch:
		// Only delete if branch is tracked
		if eng.IsBranchTracked(step.BranchName) {
			if err := eng.DeleteBranch(ctx, step.BranchName); err != nil {
				// Non-fatal - branch might already be deleted
				splog.Debug("Failed to delete branch %s (may already be deleted): %v", step.BranchName, err)
			}
		}

	case StepUpdatePRBase:
		// For top-down strategy: rebase branch onto trunk and update PR base
		if err := executeUpdatePRBase(ctx, eng, githubClient, step); err != nil {
			return err
		}

	case StepWaitCI:
		// StepWaitCI should be handled by executeStepWithProgress, not executeStep
		// This case should never be reached, but if it is, return an error
		return fmt.Errorf("StepWaitCI should be handled by executeStepWithProgress")

	default:
		return fmt.Errorf("unknown step type: %s", step.StepType)
	}

	return nil
}

// executeUpdatePRBase handles the UPDATE_PR_BASE step
// This is used in top-down strategy to rebase the current branch onto trunk
func executeUpdatePRBase(ctx context.Context, eng mergeExecuteEngine, githubClient github.Client, step PlanStep) error {
	trunk := eng.Trunk()

	// Get the parent revision (old base)
	parent := eng.GetParent(step.BranchName)
	if parent == "" {
		parent = trunk
	}

	// Get the old parent revision
	oldParentRev, err := eng.GetRevision(ctx, parent)
	if err != nil {
		return fmt.Errorf("failed to get parent revision: %w", err)
	}

	// If parent is already trunk, we might just need to update the PR base
	if parent == trunk {
		// Just update the PR base branch via GitHub API
		return updatePRBaseBranchFromContext(ctx, githubClient, step.BranchName, trunk)
	}

	// Rebase the branch onto trunk
	gitResult, err := git.Rebase(ctx, step.BranchName, trunk, oldParentRev)
	if err != nil {
		return fmt.Errorf("failed to rebase: %w", err)
	}

	if gitResult == git.RebaseConflict {
		return fmt.Errorf("rebase conflict while rebasing %s onto %s", step.BranchName, trunk)
	}

	// Update parent to trunk
	if err := eng.SetParent(ctx, step.BranchName, trunk); err != nil {
		return fmt.Errorf("failed to update parent: %w", err)
	}

	// Update PR base branch via GitHub API
	if err := updatePRBaseBranchFromContext(ctx, githubClient, step.BranchName, trunk); err != nil {
		return fmt.Errorf("failed to update PR base: %w", err)
	}

	// Rebuild engine to reflect changes
	if err := eng.Rebuild(trunk); err != nil {
		return fmt.Errorf("failed to rebuild engine: %w", err)
	}

	return nil
}

// updatePRBaseBranchFromContext updates a PR's base branch via GitHub API
func updatePRBaseBranchFromContext(ctx context.Context, githubClient github.Client, branchName, newBase string) error {
	if githubClient == nil {
		// If we can't get GitHub client, skip this step (non-fatal)
		return nil
	}

	owner, repo := githubClient.GetOwnerRepo()

	// Get PR for this branch
	pr, err := githubClient.GetPullRequestByBranch(ctx, owner, repo, branchName)
	if err != nil || pr == nil {
		// PR not found or error - non-fatal
		return nil //nolint:nilerr
	}

	// Update PR base
	updateOpts := github.UpdatePROptions{
		Base: &newBase,
	}

	if err := githubClient.UpdatePullRequest(ctx, owner, repo, pr.Number, updateOpts); err != nil {
		return fmt.Errorf("failed to update PR base: %w", err)
	}

	return nil
}

// executeWaitCIWithProgress waits for CI checks with progress reporting
func executeWaitCIWithProgress(ctx context.Context, step PlanStep, stepIndex int, eng mergeExecuteEngine, splog *tui.Splog, githubClient github.Client, opts ExecuteOptions) error {
	if githubClient == nil {
		return fmt.Errorf("GitHub client not available")
	}

	timeout := step.WaitTimeout
	if timeout == 0 {
		timeout = 10 * time.Minute // Default timeout
	}

	pollInterval := 15 * time.Second    // Poll every 15 seconds
	progressInterval := 1 * time.Second // Report progress every second
	startTime := time.Now()
	deadline := startTime.Add(timeout)
	lastProgressReport := startTime

	// Get PR info for better error messages
	prInfo, err := eng.GetPrInfo(step.BranchName)
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}
	prNumber := step.PRNumber
	if prInfo != nil && prInfo.Number != nil {
		prNumber = *prInfo.Number
	}

	if opts.Reporter == nil {
		splog.Info("Waiting for CI checks on PR #%d (%s)...", prNumber, step.BranchName)
	}

	for {
		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for CI checks on PR #%d (%s) after %v", prNumber, step.BranchName, timeout)
		}

		// Report progress periodically
		now := time.Now()
		if opts.Reporter != nil && now.Sub(lastProgressReport) >= progressInterval {
			elapsed := now.Sub(startTime)
			opts.Reporter.StepWaiting(stepIndex, elapsed, timeout)
			lastProgressReport = now
		}

		// Check CI status
		passing, pending, err := githubClient.GetPRChecksStatus(ctx, step.BranchName)
		if err != nil {
			// Log error but continue polling (might be transient)
			splog.Debug("Error checking CI status: %v", err)
		} else {
			if !passing {
				// CI checks failed
				return fmt.Errorf("CI checks failed on PR #%d (%s)", prNumber, step.BranchName)
			}
			if !pending {
				// All checks passed and none are pending
				elapsed := time.Since(startTime)
				if opts.Reporter == nil {
					splog.Info("CI checks passed on PR #%d (%s) after %v", prNumber, step.BranchName, elapsed.Round(time.Second))
				}
				return nil
			}
		}

		// Wait before next poll
		remaining := time.Until(deadline)
		if remaining < pollInterval {
			time.Sleep(remaining)
		} else {
			time.Sleep(pollInterval)
		}
	}
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
