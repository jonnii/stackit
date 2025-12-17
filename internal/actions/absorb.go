package actions

import (
	"fmt"
	"sort"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// AbsorbOptions contains options for the absorb command
type AbsorbOptions struct {
	All    bool
	DryRun bool
	Force  bool
	Patch  bool
}

// AbsorbAction performs the absorb operation
func AbsorbAction(ctx *runtime.Context, opts AbsorbOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Check if rebase is in progress
	if err := checkRebaseInProgress(); err != nil {
		return err
	}

	// Check if there are staged changes (before handling flags)
	hasStaged, err := git.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	// Handle --all flag: stage all tracked changes
	if opts.All {
		if err := git.StageTracked(); err != nil {
			return fmt.Errorf("failed to stage all changes: %w", err)
		}
	}

	// Handle --patch flag: interactive patch staging
	if opts.Patch {
		if err := git.StagePatch(); err != nil {
			return fmt.Errorf("failed to stage patch: %w", err)
		}
	}

	// Re-check staged changes after flags
	hasStaged, err = git.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStaged {
		return fmt.Errorf("no staged changes to absorb. Use --all to stage all changes or --patch to interactively stage changes")
	}

	// Parse staged hunks
	hunks, err := git.ParseStagedHunks()
	if err != nil {
		return fmt.Errorf("failed to parse staged hunks: %w", err)
	}

	if len(hunks) == 0 {
		return fmt.Errorf("no hunks found in staged changes")
	}

	// Get all commits downstack from current branch
	// We need commits from all branches downstack, not just current branch
	scope := engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: false,
	}
	downstackBranches := eng.GetRelativeStack(currentBranch, scope)

	// Get all commit SHAs from downstack branches (newest to oldest)
	commitSHAs, err := getAllCommitSHAs(downstackBranches, eng)
	if err != nil {
		return fmt.Errorf("failed to get commit SHAs: %w", err)
	}

	// Find target commit for each hunk
	hunkTargets := []git.HunkTarget{}
	unabsorbedHunks := []git.Hunk{}

	for _, hunk := range hunks {
		commitSHA, commitIndex, err := git.FindTargetCommitForHunk(hunk, commitSHAs)
		if err != nil {
			return fmt.Errorf("failed to find target commit for hunk: %w", err)
		}

		if commitSHA == "" {
			// Hunk commutes with all commits - can't be absorbed
			unabsorbedHunks = append(unabsorbedHunks, hunk)
			continue
		}

		hunkTargets = append(hunkTargets, git.HunkTarget{
			Hunk:        hunk,
			CommitSHA:   commitSHA,
			CommitIndex: commitIndex,
		})
	}

	// Group hunks by commit
	hunksByCommit := make(map[string][]git.Hunk)
	for _, target := range hunkTargets {
		hunksByCommit[target.CommitSHA] = append(hunksByCommit[target.CommitSHA], target.Hunk)
	}

	// Print dry-run output or confirmation
	if opts.DryRun {
		return printDryRunOutput(hunksByCommit, unabsorbedHunks, downstackBranches, eng, splog)
	}

	// Print what will be absorbed
	if err := printAbsorbPlan(hunksByCommit, unabsorbedHunks, downstackBranches, eng, splog); err != nil {
		return err
	}

	// Prompt for confirmation if not --force
	if !opts.Force {
		confirmed, err := tui.PromptConfirm("Apply these changes to the commits?", false)
		if err != nil {
			return fmt.Errorf("confirmation cancelled: %w", err)
		}
		if !confirmed {
			splog.Info("Absorb cancelled")
			return nil
		}
	}

	// Apply hunks to commits
	// Process commits from oldest to newest to avoid conflicts
	commitList := make([]string, 0, len(hunksByCommit))
	for commitSHA := range hunksByCommit {
		commitList = append(commitList, commitSHA)
	}
	sort.Slice(commitList, func(i, j int) bool {
		// Sort by commit index (oldest first)
		idxI := -1
		idxJ := -1
		for _, target := range hunkTargets {
			if target.CommitSHA == commitList[i] {
				idxI = target.CommitIndex
			}
			if target.CommitSHA == commitList[j] {
				idxJ = target.CommitIndex
			}
		}
		return idxI > idxJ // Higher index = older commit
	})

	for _, commitSHA := range commitList {
		hunks := hunksByCommit[commitSHA]

		// Find branch for this commit
		branchName, err := findBranchForCommit(commitSHA, downstackBranches, eng)
		if err != nil {
			return fmt.Errorf("failed to find branch for commit: %w", err)
		}

		// Apply all hunks for this commit together
		if err := git.ApplyHunksToCommit(hunks, commitSHA, branchName); err != nil {
			return fmt.Errorf("failed to apply hunks to commit %s: %w", commitSHA[:8], err)
		}

		splog.Info("Absorbed changes into commit %s in %s", commitSHA[:8], tui.ColorBranchName(branchName, false))
	}

	// Warn about unabsorbed hunks
	if len(unabsorbedHunks) > 0 {
		splog.Warn("The following hunks could not be absorbed (they commute with all commits):")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	// Restack upstack branches
	scope = engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := eng.GetRelativeStack(currentBranch, scope)

	if len(upstackBranches) > 0 {
		if err := RestackBranches(upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}

// getAllCommitSHAs gets all commit SHAs from a list of branches (newest to oldest)
func getAllCommitSHAs(branches []string, eng engine.Engine) ([]string, error) {
	var allSHAs []string

	for _, branchName := range branches {
		// Get commits for this branch
		commits, err := eng.GetAllCommits(branchName, engine.CommitFormatSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to get commits for branch %s: %w", branchName, err)
		}

		// Commits are returned oldest to newest, but we want newest to oldest
		for i := len(commits) - 1; i >= 0; i-- {
			allSHAs = append(allSHAs, commits[i])
		}
	}

	return allSHAs, nil
}

// findBranchForCommit finds which branch a commit belongs to
func findBranchForCommit(commitSHA string, branches []string, eng engine.Engine) (string, error) {
	for _, branchName := range branches {
		commits, err := eng.GetAllCommits(branchName, engine.CommitFormatSHA)
		if err != nil {
			continue
		}

		for _, sha := range commits {
			if sha == commitSHA {
				return branchName, nil
			}
		}
	}

	return "", fmt.Errorf("commit %s not found in any branch", commitSHA)
}

// printDryRunOutput prints what would be absorbed in dry-run mode
func printDryRunOutput(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, branches []string, eng engine.Engine, splog *tui.Splog) error {
	splog.Info("Would absorb the following changes:")
	splog.Newline()

	// Get commit info for display
	for commitSHA, hunks := range hunksByCommit {
		branchName, err := findBranchForCommit(commitSHA, branches, eng)
		if err != nil {
			branchName = "unknown"
		}

		// Get commit message - show first commit message from the branch
		commits, err := eng.GetAllCommits(branchName, engine.CommitFormatReadable)
		if err == nil && len(commits) > 0 {
			splog.Info("  %s in %s:", commitSHA[:8], tui.ColorBranchName(branchName, false))
			splog.Info("    %s", commits[0])
		} else {
			splog.Info("  %s in %s:", commitSHA[:8], tui.ColorBranchName(branchName, false))
		}

		for _, hunk := range hunks {
			splog.Info("    - %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	if len(unabsorbedHunks) > 0 {
		splog.Newline()
		splog.Warn("The following hunks would not be absorbed:")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	return nil
}

// printAbsorbPlan prints the plan for absorbing changes
func printAbsorbPlan(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, branches []string, eng engine.Engine, splog *tui.Splog) error {
	splog.Info("Will absorb the following changes:")
	splog.Newline()

	for commitSHA, hunks := range hunksByCommit {
		branchName, err := findBranchForCommit(commitSHA, branches, eng)
		if err != nil {
			branchName = "unknown"
		}

		splog.Info("  Commit %s in %s:", commitSHA[:8], tui.ColorBranchName(branchName, false))
		for _, hunk := range hunks {
			splog.Info("    - %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	if len(unabsorbedHunks) > 0 {
		splog.Newline()
		splog.Warn("The following hunks will not be absorbed:")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	return nil
}

// checkRebaseInProgress checks if a rebase is in progress
func checkRebaseInProgress() error {
	if git.IsRebaseInProgress() {
		return fmt.Errorf("cannot absorb during a rebase. Please finish or abort the rebase first")
	}
	return nil
}
