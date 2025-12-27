package actions

import (
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
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
		currentBranch := eng.CurrentBranch()
		if currentBranch == nil {
			return fmt.Errorf("not on a branch and no branch specified")
		}
		branchName = currentBranch.GetName()
	}

	branch := eng.GetBranch(branchName)

	if !branch.IsTracked() && !branch.IsTrunk() {
		_, err := eng.GetRevisionInternal(branchName)
		if err != nil {
			return fmt.Errorf("branch %s does not exist", branchName)
		}
	}

	// If stat is set without diff or patch, it implies diff
	effectiveDiff := opts.Diff || (opts.Stat && !opts.Patch)
	effectivePatch := opts.Patch && !opts.Diff

	var outputLines []string

	currentBranch := eng.CurrentBranch()
	isCurrent := branchName == currentBranch.GetName()
	isTrunk := branch.IsTrunk()

	coloredBranchName := style.ColorBranchName(branchName, isCurrent)

	if !isTrunk && !branch.IsBranchUpToDate() {
		coloredBranchName += " " + style.ColorNeedsRestack("(needs restack)")
	}
	outputLines = append(outputLines, coloredBranchName)

	commitDate, err := branch.GetCommitDate()
	if err == nil {
		dateStr := commitDate.Format(time.RFC3339)
		outputLines = append(outputLines, style.ColorDim(dateStr))
	}

	var prInfo *engine.PrInfo
	if !isTrunk {
		branch := eng.GetBranch(branchName)
		prInfo, _ = eng.GetPrInfo(branch)
		if prInfo != nil && prInfo.Number() != nil {
			prTitleLine := getPRTitleLine(prInfo)
			if prTitleLine != "" {
				outputLines = append(outputLines, "")
				outputLines = append(outputLines, prTitleLine)
			}
			if prInfo.URL() != "" {
				outputLines = append(outputLines, style.ColorMagenta(prInfo.URL()))
			}
		}
	}

	branchObj := eng.GetBranch(branchName)
	parentBranch := eng.GetParent(branchObj)
	if parentBranch != nil {
		outputLines = append(outputLines, "")
		outputLines = append(outputLines, fmt.Sprintf("%s: %s", style.ColorCyan("Parent"), parentBranch.GetName()))
	}

	children := branchObj.GetChildren()
	if len(children) > 0 {
		outputLines = append(outputLines, fmt.Sprintf("%s:", style.ColorCyan("Children")))
		for _, child := range children {
			outputLines = append(outputLines, fmt.Sprintf("▸ %s", child.GetName()))
		}
	}

	if opts.Body && prInfo != nil && prInfo.Body() != "" {
		outputLines = append(outputLines, "")
		outputLines = append(outputLines, prInfo.Body())
	}

	outputLines = append(outputLines, "")
	if effectivePatch {
		baseRevision := ""
		if isTrunk {
			baseRevision = branchName + "~"
		} else {
			commits, err := branchObj.GetAllCommits(engine.CommitFormatSHA)
			if err == nil && len(commits) > 0 {
				oldestSHA := commits[0]
				baseRevision, _ = eng.GetParentCommitSHA(oldestSHA)
			}
		}
		branchRevision, err := branch.GetRevision()
		if err == nil {
			commitsOutput, err := eng.ShowCommits(ctx.Context, baseRevision, branchRevision, true, opts.Stat)
			if err == nil && commitsOutput != "" {
				outputLines = append(outputLines, commitsOutput)
			}
		}
	} else {
		commits, err := branch.GetAllCommits(engine.CommitFormatReadable)
		if err == nil {
			for _, commit := range commits {
				outputLines = append(outputLines, style.ColorDim(commit))
			}
		}
	}

	if effectiveDiff {
		outputLines = append(outputLines, "")
		if isTrunk {
			headRevision, err := branch.GetRevision()
			if err == nil {
				parentSHA, err := eng.GetCommitSHA(branchName, 1)
				if err == nil {
					diffOutput, err := eng.ShowDiff(ctx.Context, parentSHA, headRevision, opts.Stat)
					if err == nil && diffOutput != "" {
						outputLines = append(outputLines, diffOutput)
					}
				}
			}
		} else {
			commits, err := branchObj.GetAllCommits(engine.CommitFormatSHA)
			if err == nil && len(commits) > 0 {
				oldestSHA := commits[0]
				parentSHA, _ := eng.GetParentCommitSHA(oldestSHA)
				branchRevision, err := branch.GetRevision()
				if err == nil {
					diffOutput, err := eng.ShowDiff(ctx.Context, parentSHA, branchRevision, opts.Stat)
					if err == nil && diffOutput != "" {
						outputLines = append(outputLines, diffOutput)
					}
				}
			}
		}
	}

	// Apply dimming for merged/closed PRs
	const (
		prStateMerged = "MERGED"
		prStateClosed = "CLOSED"
	)
	if prInfo != nil && (prInfo.State() == prStateMerged || prInfo.State() == prStateClosed) {
		for i := range outputLines {
			outputLines[i] = style.ColorDim(outputLines[i])
		}
	}

	splog.Page(strings.Join(outputLines, "\n"))
	splog.Newline()

	return nil
}

func getPRTitleLine(prInfo *engine.PrInfo) string {
	if prInfo == nil || prInfo.Number() == nil || prInfo.Title() == "" {
		return ""
	}

	state := prInfo.State()

	const (
		prStateMerged = "MERGED"
		prStateClosed = "CLOSED"
	)

	prNumber := style.ColorPRNumber(*prInfo.Number())

	switch state {
	case prStateMerged:
		return fmt.Sprintf("%s (Merged) %s", prNumber, prInfo.Title())
	case prStateClosed:
		return fmt.Sprintf("%s (Abandoned) %s", prNumber, style.ColorDim(prInfo.Title()))
	default:
		prState := style.ColorPRState(state, prInfo.IsDraft())
		return fmt.Sprintf("%s %s %s", prNumber, prState, prInfo.Title())
	}
}
