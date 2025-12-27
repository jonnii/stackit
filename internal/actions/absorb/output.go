package absorb

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

const unknown = "unknown"

// printDryRunOutput prints what would be absorbed in dry-run mode
func printDryRunOutput(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, eng engine.Engine, splog *tui.Splog) {
	splog.Info("Would absorb the following changes:")
	splog.Newline()

	// Get commit info for display
	for commitSHA, hunks := range hunksByCommit {
		branchName, err := eng.FindBranchForCommit(commitSHA)
		if err != nil {
			branchName = unknown
		}

		// Get commit message - show first commit message from the branch
		branch := eng.GetBranch(branchName)
		commits, err := branch.GetAllCommits(engine.CommitFormatReadable)
		if err == nil && len(commits) > 0 {
			splog.Info("  %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
			splog.Info("    %s", commits[0])
		} else {
			splog.Info("  %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
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
}

// printAbsorbPlan prints the plan for absorbing changes
func printAbsorbPlan(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, eng engine.Engine, splog *tui.Splog) {
	splog.Info("Will absorb the following changes:")
	splog.Newline()

	for commitSHA, hunks := range hunksByCommit {
		branchName, err := eng.FindBranchForCommit(commitSHA)
		if err != nil {
			branchName = unknown
		}

		splog.Info("  Commit %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
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
}
