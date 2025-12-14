package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// MergeOptions are options for the merge command
type MergeOptions struct {
	DryRun  bool
	Confirm bool
	Engine  engine.Engine
	Splog   *output.Splog
}

// MergeAction performs the merge operation
func MergeAction(opts MergeOptions) error {
	eng := opts.Engine
	splog := opts.Splog

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Check if current branch is trunk
	if eng.IsTrunk(currentBranch) {
		return fmt.Errorf("cannot merge from trunk. You must be on a branch that has a PR")
	}

	// Check if current branch is tracked
	if !eng.IsBranchTracked(currentBranch) {
		return fmt.Errorf("current branch %s is not tracked by stackit", currentBranch)
	}

	// Get all branches from trunk to current branch
	scope := engine.Scope{RecursiveParents: true}
	parentBranches := eng.GetRelativeStack(currentBranch, scope)
	
	// Build full list: parent branches + current branch
	allBranches := make([]string, 0, len(parentBranches)+1)
	allBranches = append(allBranches, parentBranches...)
	allBranches = append(allBranches, currentBranch)

	// Collect PRs to merge
	type prToMerge struct {
		branchName string
		prNumber   int
		prURL      string
		differs    bool
	}

	prsToMerge := []prToMerge{}

	for _, branchName := range allBranches {
		// Get PR info
		prInfo, err := eng.GetPrInfo(branchName)
		if err != nil {
			splog.Debug("Failed to get PR info for %s: %v", branchName, err)
			continue
		}

		// Skip if no PR
		if prInfo == nil || prInfo.Number == nil {
			splog.Debug("No PR found for branch %s", branchName)
			continue
		}

		// Skip if PR is not open
		state := strings.ToUpper(prInfo.State)
		if state != "OPEN" {
			splog.Info("Skipping %s: PR #%d is %s", branchName, *prInfo.Number, state)
			continue
		}

		// Check if local branch differs from remote
		matchesRemote, err := eng.BranchMatchesRemote(branchName)
		if err != nil {
			splog.Debug("Failed to check if branch matches remote: %v", err)
			matchesRemote = true // Assume matches if check fails
		}

		prsToMerge = append(prsToMerge, prToMerge{
			branchName: branchName,
			prNumber:   *prInfo.Number,
			prURL:      prInfo.URL,
			differs:    !matchesRemote,
		})
	}

	// If no PRs to merge, exit early
	if len(prsToMerge) == 0 {
		splog.Info("No open PRs found to merge")
		return nil
	}

	// Dry run mode: just report what would be merged
	if opts.DryRun {
		splog.Info("Would merge the following PRs:")
		splog.Newline()
		for _, pr := range prsToMerge {
			branchColor := output.ColorBranchName(pr.branchName, false)
			splog.Info("  %s: PR #%d", branchColor, pr.prNumber)
			if pr.prURL != "" {
				splog.Info("    %s", output.ColorDim(pr.prURL))
			}
		}
		return nil
	}

	// Check if we need confirmation
	needsConfirmation := opts.Confirm
	for _, pr := range prsToMerge {
		if pr.differs {
			needsConfirmation = true
			break
		}
	}

	// Prompt for confirmation if needed
	if needsConfirmation {
		splog.Info("The following PRs will be merged:")
		splog.Newline()
		for _, pr := range prsToMerge {
			branchColor := output.ColorBranchName(pr.branchName, false)
			differsMsg := ""
			if pr.differs {
				differsMsg = output.ColorDim(" (local differs from remote)")
			}
			splog.Info("  %s: PR #%d%s", branchColor, pr.prNumber, differsMsg)
			if pr.prURL != "" {
				splog.Info("    %s", output.ColorDim(pr.prURL))
			}
		}
		splog.Newline()

		confirmed, err := promptConfirm("Merge these PRs?", false)
		if err != nil {
			return fmt.Errorf("confirmation cancelled: %w", err)
		}
		if !confirmed {
			splog.Info("Merge cancelled")
			return nil
		}
	}

	// Merge each PR
	for _, pr := range prsToMerge {
		splog.Info("Merging PR #%d for %s...", pr.prNumber, output.ColorBranchName(pr.branchName, false))
		
		if err := git.MergePullRequest(pr.branchName); err != nil {
			splog.Warn("Failed to merge PR #%d for %s: %v", pr.prNumber, pr.branchName, err)
			// Continue with other PRs
			continue
		}

		splog.Info("Successfully merged PR #%d for %s", pr.prNumber, output.ColorBranchName(pr.branchName, true))
	}

	return nil
}
