package actions

import (
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// InfoOptions contains options for the info command
type InfoOptions struct {
	BranchName string
	Body       bool
	Diff       bool
	Patch      bool
	Stat       bool
}

// InfoAction displays information about a branch
func InfoAction(ctx *runtime.Context, opts InfoOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branchName := opts.BranchName
	if branchName == "" {
		branchName = eng.CurrentBranch()
		if branchName == "" {
			return fmt.Errorf("not on a branch and no branch specified")
		}
	}

	// Check if branch exists
	if !eng.IsBranchTracked(branchName) && !eng.IsTrunk(branchName) {
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
	isCurrent := branchName == eng.CurrentBranch()
	isTrunk := eng.IsTrunk(branchName)

	// Branch name with current indicator
	coloredBranchName := tui.ColorBranchName(branchName, isCurrent)

	// Add restack indicator if needed
	if !isTrunk && !eng.IsBranchFixed(branchName) {
		coloredBranchName += " " + tui.ColorNeedsRestack("(needs restack)")
	}
	outputLines = append(outputLines, coloredBranchName)

	// Commit date
	commitDate, err := eng.GetCommitDate(branchName)
	if err == nil {
		dateStr := commitDate.Format(time.RFC3339)
		outputLines = append(outputLines, tui.ColorDim(dateStr))
	}

	// PR info (skip for trunk)
	var prInfo *engine.PrInfo
	if !isTrunk {
		prInfo, _ = eng.GetPrInfo(branchName)
		if prInfo != nil && prInfo.Number != nil {
			prTitleLine := getPRTitleLine(prInfo)
			if prTitleLine != "" {
				outputLines = append(outputLines, "")
				outputLines = append(outputLines, prTitleLine)
			}
			if prInfo.URL != "" {
				outputLines = append(outputLines, tui.ColorMagenta(prInfo.URL))
			}
		}
	}

	// Parent branch
	parentBranchName := eng.GetParent(branchName)
	if parentBranchName != "" {
		outputLines = append(outputLines, "")
		outputLines = append(outputLines, fmt.Sprintf("%s: %s", tui.ColorCyan("Parent"), parentBranchName))
	}

	// Children branches
	children := eng.GetChildren(branchName)
	if len(children) > 0 {
		outputLines = append(outputLines, fmt.Sprintf("%s:", tui.ColorCyan("Children")))
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
		branchRevision, err := eng.GetRevision(branchName)
		if err == nil {
			commitsOutput, err := git.ShowCommits(baseRevision, branchRevision, true, opts.Stat)
			if err == nil && commitsOutput != "" {
				outputLines = append(outputLines, commitsOutput)
			}
		}
	} else {
		// Show commit list (readable format)
		commits, err := eng.GetAllCommits(branchName, engine.CommitFormatReadable)
		if err == nil {
			for _, commit := range commits {
				outputLines = append(outputLines, tui.ColorDim(commit))
			}
		}
	}

	// Diff (if requested and not showing patches)
	if effectiveDiff {
		outputLines = append(outputLines, "")
		if isTrunk {
			// For trunk, show diff from HEAD~1 to HEAD
			headRevision, err := eng.GetRevision(branchName)
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
				branchRevision, err := eng.GetRevision(branchName)
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
			outputLines[i] = tui.ColorDim(outputLines[i])
		}
	}

	// Output the result
	splog.Page(strings.Join(outputLines, "\n"))
	splog.Newline()

	return nil
}

// getPRTitleLine formats the PR title line with number, state, and title
func getPRTitleLine(prInfo *engine.PrInfo) string {
	if prInfo == nil || prInfo.Number == nil || prInfo.Title == "" {
		return ""
	}

	prNumber := tui.ColorPRNumber(*prInfo.Number)
	state := prInfo.State

	if state == "MERGED" {
		return fmt.Sprintf("%s (Merged) %s", prNumber, prInfo.Title)
	} else if state == "CLOSED" {
		// Strikethrough not easily available, use dim instead
		return fmt.Sprintf("%s (Abandoned) %s", prNumber, tui.ColorDim(prInfo.Title))
	} else {
		prState := tui.ColorPRState(state, prInfo.IsDraft)
		return fmt.Sprintf("%s %s %s", prNumber, prState, prInfo.Title)
	}
}
