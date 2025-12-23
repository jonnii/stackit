package split

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// Style specifies the split mode
type Style string

const (
	// StyleCommit splits by selecting commit points
	StyleCommit Style = "commit"
	// StyleHunk splits by interactively staging hunks
	StyleHunk Style = "hunk"
	// StyleFile splits by extracting specified files
	StyleFile Style = "file"
)

// Options contains options for the split command
type Options struct {
	Style     Style
	Pathspecs []string
}

// Result contains the result of a split operation
type Result struct {
	BranchNames  []string // From oldest to newest
	BranchPoints []int    // Commit indices (0 = HEAD, 1 = HEAD~1, etc.)
}

// Action performs the split operation
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog
	context := ctx.Context

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	// Check for uncommitted tracked changes
	hasUnstaged, err := git.HasUnstagedChanges(context)
	if err != nil {
		return fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	if hasUnstaged {
		return fmt.Errorf("cannot split: you have uncommitted tracked changes")
	}

	// Ensure branch is tracked
	currentBranchObj := eng.GetBranch(currentBranch.Name)
	if !currentBranchObj.IsTracked() {
		// Auto-track the branch
		parent := eng.GetParent(*currentBranch)
		parentName := ""
		if parent == nil {
			// Try to find parent from git
			parentName = eng.Trunk().Name
		} else {
			parentName = parent.Name
		}
		if err := eng.TrackBranch(context, currentBranch.Name, parentName); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}
	}

	// Determine style
	style := opts.Style
	if style == "" {
		// Check if there's more than one commit
		commits, err := currentBranch.GetAllCommits(engine.CommitFormatSHA)
		if err != nil {
			return fmt.Errorf("failed to get commits: %w", err)
		}

		if len(commits) > 1 {
			// Prompt for style
			var styleStr string
			prompt := &survey.Select{
				Message: fmt.Sprintf("How would you like to split %s?", currentBranch.Name),
				Options: []string{"By commit - slice up the history of this branch", "By hunk - split into new single-commit branches", "Cancel"},
			}
			if err := survey.AskOne(prompt, &styleStr); err != nil {
				return fmt.Errorf("canceled")
			}

			switch {
			case strings.Contains(styleStr, "Cancel"):
				return fmt.Errorf("canceled")
			case strings.Contains(styleStr, "commit"):
				style = StyleCommit
			case strings.Contains(styleStr, "hunk"):
				style = StyleHunk
			}
		} else {
			// Only one commit, default to hunk
			style = StyleHunk
		}
	}

	// Perform the split
	var result *Result
	switch style {
	case StyleCommit:
		result, err = splitByCommit(context, currentBranch.Name, eng, splog)
	case StyleHunk:
		result, err = splitByHunk(context, currentBranch.Name, eng, splog)
	case StyleFile:
		pathspecs := opts.Pathspecs
		// If no pathspecs provided, prompt interactively
		if len(pathspecs) == 0 {
			pathspecs, err = promptForFiles(context, currentBranch.Name, eng, splog)
			if err != nil {
				return err
			}
			if len(pathspecs) == 0 {
				return fmt.Errorf("no files selected")
			}
		}
		// splitByFile handles everything internally (creating branches, tracking, etc.)
		// and updates the parent relationship, so we just need to restack upstack branches
		_, err = splitByFile(context, currentBranch.Name, pathspecs, eng)
		if err != nil {
			return err
		}
		// Restack upstack branches if any
		rng := engine.StackRange{
			RecursiveParents:  false,
			IncludeCurrent:    false,
			RecursiveChildren: true,
		}
		upstackBranches := currentBranch.GetRelativeStack(rng)
		if len(upstackBranches) > 0 {
			if err := actions.RestackBranches(context, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
				return fmt.Errorf("failed to restack upstack branches: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown split style: %s", style)
	}

	if err != nil {
		return err
	}

	// Get upstack branches (children)
	rng := engine.StackRange{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := currentBranch.GetRelativeStack(rng)

	// Apply the split
	if err := eng.ApplySplitToCommits(context, engine.ApplySplitOptions{
		BranchToSplit: currentBranch.Name,
		BranchNames:   result.BranchNames,
		BranchPoints:  result.BranchPoints,
	}); err != nil {
		return fmt.Errorf("failed to apply split: %w", err)
	}

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := actions.RestackBranches(context, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
