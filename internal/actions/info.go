package actions

import (
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// InfoOptions specifies options for the info command
type InfoOptions struct {
	BranchName string
	Body       bool
	Diff       bool
	Patch      bool
	Stat       bool
	Engine     engine.Engine
	Splog      *output.Splog
}

// InfoAction displays information about a branch
func InfoAction(opts InfoOptions) error {
	branchName := opts.BranchName
	if branchName == "" {
		branchName = opts.Engine.CurrentBranch()
		if branchName == "" {
			return fmt.Errorf("not on a branch and no branch specified")
		}
	}

	// Check if branch exists
	if !opts.Engine.IsBranchTracked(branchName) && !opts.Engine.IsTrunk(branchName) {
		// Check if it's a git branch
		_, err := git.GetRevision(branchName)
		if err != nil {
			return fmt.Errorf("branch %s does not exist", branchName)
		}
	}

	// Determine effective flags
	// If stat is set without diff or patch, it implies diff
	effectiveDiff := opts.Diff || (opts.Stat && !opts.Patch)
	effectivePatch := opts.Patch && !opts.Diff

	// Build output lines
	var outputLines []string

	// Get branch info
	isCurrent := branchName == opts.Engine.CurrentBranch()
	isTrunk := opts.Engine.IsTrunk(branchName)

	// Branch name with current indicator
	coloredBranchName := output.ColorBranchName(branchName, isCurrent)

	// Add restack indicator if needed
	if !isTrunk && !opts.Engine.IsBranchFixed(branchName) {
		coloredBranchName += " " + output.ColorNeedsRestack("(needs restack)")
	}
	outputLines = append(outputLines, coloredBranchName)

	// Commit date
	commitDate, err := opts.Engine.GetCommitDate(branchName)
	if err == nil {
		dateStr := commitDate.Format(time.RFC3339)
		outputLines = append(outputLines, output.ColorDim(dateStr))
	}

	// PR info (skip for trunk)
	var prInfo *engine.PrInfo
	if !isTrunk {
		prInfo, _ = opts.Engine.GetPrInfo(branchName)
		if prInfo != nil && prInfo.Number != nil {
			prTitleLine := getPRTitleLine(prInfo)
			if prTitleLine != "" {
				outputLines = append(outputLines, "")
				outputLines = append(outputLines, prTitleLine)
			}
			if prInfo.URL != "" {
				outputLines = append(outputLines, output.ColorMagenta(prInfo.URL))
			}
		}
	}

	// Parent branch
	parentBranchName := opts.Engine.GetParent(branchName)
	if parentBranchName != "" {
		outputLines = append(outputLines, "")
		outputLines = append(outputLines, fmt.Sprintf("%s: %s", output.ColorCyan("Parent"), parentBranchName))
	}

	// Children branches
	children := opts.Engine.GetChildren(branchName)
	if len(children) > 0 {
		outputLines = append(outputLines, fmt.Sprintf("%s:", output.ColorCyan("Children")))
		for _, child := range children {
			outputLines = append(outputLines, fmt.Sprintf("▸ %s", child))
		}
	}

	// PR body
	if opts.Body && prInfo != nil && prInfo.Body != "" {
		outputLines = append(outputLines, "")
		outputLines = append(outputLines, prInfo.Body)
	}

	// Commits listing
	outputLines = append(outputLines, "")
	if effectivePatch {
		// Show commits with patches
		baseRevision := ""
		if isTrunk {
			// For trunk, use parent commit (branchName~)
			baseRevision = branchName + "~"
		} else {
			meta, err := git.ReadMetadataRef(branchName)
			if err == nil && meta.ParentBranchRevision != nil {
				baseRevision = *meta.ParentBranchRevision
			}
		}
		branchRevision, err := opts.Engine.GetRevision(branchName)
		if err == nil {
			commitsOutput, err := git.ShowCommits(baseRevision, branchRevision, true, opts.Stat)
			if err == nil && commitsOutput != "" {
				outputLines = append(outputLines, commitsOutput)
			}
		}
	} else {
		// Show commit list (readable format)
		commits, err := opts.Engine.GetAllCommits(branchName, engine.CommitFormatReadable)
		if err == nil {
			for _, commit := range commits {
				outputLines = append(outputLines, output.ColorDim(commit))
			}
		}
	}

	// Diff (if requested and not showing patches)
	if effectiveDiff {
		outputLines = append(outputLines, "")
		if isTrunk {
			// For trunk, show diff from HEAD~1 to HEAD
			headRevision, err := opts.Engine.GetRevision(branchName)
			if err == nil {
				// Get parent commit
				parentSHA, err := git.GetCommitSHA(branchName, 1)
				if err == nil {
					diffOutput, err := git.ShowDiff(parentSHA, headRevision, opts.Stat)
					if err == nil && diffOutput != "" {
						outputLines = append(outputLines, diffOutput)
					}
				}
			}
		} else {
			// For regular branches, show diff from parent revision
			meta, err := git.ReadMetadataRef(branchName)
			if err == nil && meta.ParentBranchRevision != nil {
				branchRevision, err := opts.Engine.GetRevision(branchName)
				if err == nil {
					diffOutput, err := git.ShowDiff(*meta.ParentBranchRevision, branchRevision, opts.Stat)
					if err == nil && diffOutput != "" {
						outputLines = append(outputLines, diffOutput)
					}
				}
			}
		}
	}

	// Apply dimming for merged/closed PRs
	if prInfo != nil && (prInfo.State == "MERGED" || prInfo.State == "CLOSED") {
		for i := range outputLines {
			outputLines[i] = output.ColorDim(outputLines[i])
		}
	}

	// Output the result
	opts.Splog.Page(strings.Join(outputLines, "\n"))
	opts.Splog.Newline()

	return nil
}

// getPRTitleLine formats the PR title line with number, state, and title
func getPRTitleLine(prInfo *engine.PrInfo) string {
	if prInfo == nil || prInfo.Number == nil || prInfo.Title == "" {
		return ""
	}

	prNumber := output.ColorPRNumber(*prInfo.Number)
	state := prInfo.State

	if state == "MERGED" {
		return fmt.Sprintf("%s (Merged) %s", prNumber, prInfo.Title)
	} else if state == "CLOSED" {
		// Strikethrough not easily available, use dim instead
		return fmt.Sprintf("%s (Abandoned) %s", prNumber, output.ColorDim(prInfo.Title))
	} else {
		prState := output.ColorPRState(state, prInfo.IsDraft)
		return fmt.Sprintf("%s %s %s", prNumber, prState, prInfo.Title)
	}
}
