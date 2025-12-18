package actions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// SplitStyle specifies the split mode
type SplitStyle string

const (
	SplitStyleCommit SplitStyle = "commit"
	SplitStyleHunk   SplitStyle = "hunk"
	SplitStyleFile   SplitStyle = "file"
)

// SplitOptions contains options for the split command
type SplitOptions struct {
	Style     SplitStyle
	Pathspecs []string
}

// SplitResult contains the result of a split operation
type SplitResult struct {
	BranchNames  []string // From oldest to newest
	BranchPoints []int    // Commit indices (0 = HEAD, 1 = HEAD~1, etc.)
}

// SplitAction performs the split operation
func SplitAction(ctx *runtime.Context, opts SplitOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	context := ctx.Context

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
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
	if !eng.IsBranchTracked(currentBranch) {
		// Auto-track the branch
		parent := eng.GetParent(currentBranch)
		if parent == "" {
			// Try to find parent from git
			parent = eng.Trunk()
		}
		if err := eng.TrackBranch(context, currentBranch, parent); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}
	}

	// Determine style
	style := opts.Style
	if style == "" {
		// Check if there's more than one commit
		commits, err := eng.GetAllCommits(context, currentBranch, engine.CommitFormatSHA)
		if err != nil {
			return fmt.Errorf("failed to get commits: %w", err)
		}

		if len(commits) > 1 {
			// Prompt for style
			var styleStr string
			prompt := &survey.Select{
				Message: fmt.Sprintf("How would you like to split %s?", currentBranch),
				Options: []string{"By commit - slice up the history of this branch", "By hunk - split into new single-commit branches", "Cancel"},
			}
			if err := survey.AskOne(prompt, &styleStr); err != nil {
				return fmt.Errorf("canceled")
			}

			if strings.Contains(styleStr, "Cancel") {
				return fmt.Errorf("canceled")
			} else if strings.Contains(styleStr, "commit") {
				style = SplitStyleCommit
			} else if strings.Contains(styleStr, "hunk") {
				style = SplitStyleHunk
			}
		} else {
			// Only one commit, default to hunk
			style = SplitStyleHunk
		}
	}

	// Perform the split
	var result *SplitResult
	switch style {
	case SplitStyleCommit:
		result, err = splitByCommit(context, currentBranch, eng, splog)
	case SplitStyleHunk:
		result, err = splitByHunk(context, currentBranch, eng, splog)
	case SplitStyleFile:
		pathspecs := opts.Pathspecs
		// If no pathspecs provided, prompt interactively
		if len(pathspecs) == 0 {
			pathspecs, err = promptForFiles(context, currentBranch, eng, splog)
			if err != nil {
				return err
			}
			if len(pathspecs) == 0 {
				return fmt.Errorf("no files selected")
			}
		}
		// splitByFile handles everything internally (creating branches, tracking, etc.)
		// and updates the parent relationship, so we just need to restack upstack branches
		_, err = splitByFile(context, currentBranch, pathspecs, eng)
		if err != nil {
			return err
		}
		// Restack upstack branches if any
		scope := engine.Scope{
			RecursiveParents:  false,
			IncludeCurrent:    false,
			RecursiveChildren: true,
		}
		upstackBranches := eng.GetRelativeStack(currentBranch, scope)
		if len(upstackBranches) > 0 {
			if err := RestackBranches(context, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
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
	scope := engine.Scope{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackBranches := eng.GetRelativeStack(currentBranch, scope)

	// Apply the split
	if err := eng.ApplySplitToCommits(context, engine.ApplySplitOptions{
		BranchToSplit: currentBranch,
		BranchNames:   result.BranchNames,
		BranchPoints:  result.BranchPoints,
	}); err != nil {
		return fmt.Errorf("failed to apply split: %w", err)
	}

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := RestackBranches(context, upstackBranches, eng, splog, ctx.RepoRoot); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}

// splitByCommit splits a branch by selecting commit points
func splitByCommit(ctx context.Context, branchToSplit string, eng engine.Engine, splog *tui.Splog) (*SplitResult, error) {
	// Get readable commits
	readableCommits, err := eng.GetAllCommits(ctx, branchToSplit, engine.CommitFormatReadable)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	if len(readableCommits) == 0 {
		return nil, fmt.Errorf("no commits to split")
	}

	parentBranchName := eng.GetParentPrecondition(branchToSplit)
	numChildren := len(eng.GetChildren(branchToSplit))

	// Show instructions
	splog.Info("Splitting the commits of %s into multiple branches.", tui.ColorBranchName(branchToSplit, true))
	prInfo, _ := eng.GetPrInfo(ctx, branchToSplit)
	if prInfo != nil && prInfo.Number != nil {
		splog.Info("If any of the new branches keeps the name %s, it will be linked to PR #%d.",
			tui.ColorBranchName(branchToSplit, true), *prInfo.Number)
	}
	splog.Info("")
	splog.Info("For each branch you'd like to create:")
	splog.Info("1. Choose which commit it begins at using the below prompt.")
	splog.Info("2. Choose its name.")
	splog.Info("")

	// Get branch points interactively
	branchPoints, err := getBranchPoints(readableCommits, numChildren, parentBranchName)
	if err != nil {
		return nil, err
	}

	// Get branch names
	branchNames := []string{}
	for i := 0; i < len(branchPoints); i++ {
		splog.Info("Commits for branch %d:", i+1)
		startIdx := branchPoints[len(branchPoints)-1-i]
		var endIdx int
		if i < len(branchPoints)-1 {
			endIdx = branchPoints[len(branchPoints)-2-i]
		} else {
			endIdx = len(readableCommits)
		}
		for j := startIdx; j < endIdx; j++ {
			splog.Info("  %s", readableCommits[j])
		}
		splog.Info("")

		branchName, err := promptBranchName(ctx, branchNames, branchToSplit, i+1, eng)
		if err != nil {
			return nil, err
		}
		branchNames = append(branchNames, branchName)
	}

	// Detach HEAD to the branch revision
	branchRevision, err := eng.GetRevision(ctx, branchToSplit)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch revision: %w", err)
	}
	if err := eng.Detach(ctx, branchRevision); err != nil {
		return nil, fmt.Errorf("failed to detach: %w", err)
	}

	return &SplitResult{
		BranchNames:  branchNames,
		BranchPoints: branchPoints,
	}, nil
}

// getBranchPoints interactively gets branch points from the user
func getBranchPoints(readableCommits []string, numChildren int, parentBranchName string) ([]int, error) {
	// Array where nth index is whether we want a branch pointing to nth commit
	isBranchPoint := make([]bool, len(readableCommits))
	isBranchPoint[0] = true // First commit always has a branch

	// Build choices for the prompt
	choices := []string{}
	if numChildren > 0 {
		choices = append(choices, fmt.Sprintf("%d %s", numChildren, pluralize("child", numChildren)))
	}

	// Add commits
	for i, commit := range readableCommits {
		status := " "
		if isBranchPoint[i] {
			status = "✓"
		}
		choices = append(choices, fmt.Sprintf("%s %s", status, commit))
	}

	// Add parent and confirm
	choices = append(choices, parentBranchName)
	choices = append(choices, "Confirm")

	// Interactive loop
	for {
		var selected string
		prompt := &survey.Select{
			Message: "Toggle a commit to split the branch there. Select confirm to finish.",
			Options: choices,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return nil, fmt.Errorf("canceled")
		}

		if selected == "Confirm" {
			break
		}

		// Find the index of the selected commit
		// Choices structure: [children line (if any), commits..., parent, confirm]
		for i, choice := range choices {
			if choice == selected {
				// Skip if it's the children line, parent, or confirm
				if i == 0 && numChildren > 0 {
					// Children line - skip
					continue
				}
				if i == len(choices)-2 {
					// Parent line - skip
					continue
				}
				if i == len(choices)-1 {
					// Confirm - already handled above
					continue
				}

				// Calculate commit index
				commitIdx := i
				if numChildren > 0 {
					commitIdx-- // Skip children line
				}

				if commitIdx >= 0 && commitIdx < len(readableCommits) {
					// Never toggle the first commit
					if commitIdx != 0 {
						isBranchPoint[commitIdx] = !isBranchPoint[commitIdx]
						// Update the choice display
						status := " "
						if isBranchPoint[commitIdx] {
							status = "✓"
						}
						choices[i] = fmt.Sprintf("%s %s", status, readableCommits[commitIdx])
					}
				}
				break
			}
		}
	}

	// Convert to array of indices
	branchPoints := []int{}
	for i, isPoint := range isBranchPoint {
		if isPoint {
			branchPoints = append(branchPoints, i)
		}
	}

	return branchPoints, nil
}

// splitByHunk splits a branch by interactively staging hunks
func splitByHunk(ctx context.Context, branchToSplit string, eng engine.Engine, splog *tui.Splog) (*SplitResult, error) {
	// Detach and reset branch changes
	if err := eng.DetachAndResetBranchChanges(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to detach and reset: %w", err)
	}

	branchNames := []string{}

	// Get default commit message
	commitMessages, err := eng.GetAllCommits(ctx, branchToSplit, engine.CommitFormatMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	defaultCommitMessage := strings.Join(commitMessages, "\n\n")

	// Show instructions
	splog.Info("Splitting %s into multiple single-commit branches.", tui.ColorBranchName(branchToSplit, true))
	prInfo, _ := eng.GetPrInfo(ctx, branchToSplit)
	if prInfo != nil && prInfo.Number != nil {
		splog.Info("If any of the new branches keeps the name %s, it will be linked to PR #%d.",
			tui.ColorBranchName(branchToSplit, true), *prInfo.Number)
	}
	splog.Info("")
	splog.Info("For each branch you'd like to create:")
	splog.Info("1. Follow the prompts to stage the changes that you'd like to include.")
	splog.Info("2. Enter a commit message.")
	splog.Info("3. Pick a branch name.")
	splog.Info("The command will continue until all changes have been added to a new branch.")
	splog.Info("")

	// Loop while there are unstaged changes
	for {
		hasUnstaged, err := git.HasUnstagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check unstaged changes: %w", err)
		}
		if !hasUnstaged {
			break
		}

		// Show remaining changes
		unstagedDiff, err := git.GetUnstagedDiff(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get unstaged diff: %w", err)
		}
		splog.Info("Remaining changes:")
		splog.Info("  %s", strings.ReplaceAll(unstagedDiff, "\n", "\n  "))
		splog.Info("")

		splog.Info("Stage changes for branch %d:", len(branchNames)+1)

		// Stage patch interactively
		if err := git.StagePatch(ctx); err != nil {
			// If user cancels, restore branch
			_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
			return nil, fmt.Errorf("canceled: no new branches created")
		}

		// Check if anything was staged
		hasStaged, err := git.HasStagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStaged {
			splog.Info("No changes staged, skipping this branch.")
			continue
		}

		// Commit with message
		commitMessage := defaultCommitMessage
		var editMessage bool
		prompt := &survey.Confirm{
			Message: "Edit commit message?",
			Default: true,
		}
		if err := survey.AskOne(prompt, &editMessage); err != nil {
			return nil, fmt.Errorf("canceled")
		}

		if editMessage {
			// Get message from user
			prompt := &survey.Editor{
				Message:  "Commit message:",
				Default:  defaultCommitMessage,
				FileName: "*.md",
				Editor:   getEditor(),
			}
			if err := survey.AskOne(prompt, &commitMessage); err != nil {
				return nil, fmt.Errorf("canceled")
			}
		}

		// Create commit
		if err := git.Commit(commitMessage, 0); err != nil {
			return nil, fmt.Errorf("failed to create commit: %w", err)
		}

		// Get branch name
		branchName, err := promptBranchName(ctx, branchNames, branchToSplit, len(branchNames)+1, eng)
		if err != nil {
			return nil, err
		}
		branchNames = append(branchNames, branchName)
	}

	return &SplitResult{
		BranchNames:  branchNames,
		BranchPoints: makeRange(len(branchNames)), // Each branch is a single commit
	}, nil
}

// splitByFile splits a branch by extracting files to a new parent branch
func splitByFile(ctx context.Context, branchToSplit string, pathspecs []string, eng engine.Engine) (*SplitResult, error) {
	// Get parent branch
	parentBranchName := eng.GetParentPrecondition(branchToSplit)

	// Generate new branch name
	newBranchName := branchToSplit + "_split"
	allBranches := eng.AllBranchNames()
	for containsString(allBranches, newBranchName) {
		newBranchName = newBranchName + "_split"
	}

	// First checkout the parent branch so the new branch starts from there
	if err := git.CheckoutBranch(ctx, parentBranchName); err != nil {
		return nil, fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Create new branch from parent
	if err := git.CreateAndCheckoutBranch(ctx, newBranchName); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout files from branchToSplit
	args := append([]string{"checkout", branchToSplit, "--"}, pathspecs...)
	if _, err := git.RunGitCommandWithContext(ctx, args...); err != nil {
		// Cleanup: delete the new branch
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to checkout files: %w", err)
	}

	// Stage all changes
	if err := git.StageAll(ctx); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit
	commitMessage := fmt.Sprintf("Extract %s from %s", strings.Join(pathspecs, ", "), branchToSplit)
	if err := git.Commit(commitMessage, 0); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Track the new branch
	if err := eng.TrackBranch(ctx, newBranchName, parentBranchName); err != nil {
		_ = git.DeleteBranch(ctx, newBranchName)
		return nil, fmt.Errorf("failed to track branch: %w", err)
	}

	// Checkout original branch and remove the files
	if err := git.CheckoutBranch(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to checkout original branch: %w", err)
	}

	// Remove the files from the original branch (both index and working directory)
	args = append([]string{"rm"}, pathspecs...)
	if _, err := git.RunGitCommandWithContext(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to remove files: %w", err)
	}

	// Commit the removal
	commitMessage = fmt.Sprintf("Remove %s (moved to %s)", strings.Join(pathspecs, ", "), newBranchName)
	if err := git.Commit(commitMessage, 0); err != nil {
		return nil, fmt.Errorf("failed to commit removal: %w", err)
	}

	// Update original branch's parent to be the new split branch
	// This creates the hierarchy: parent -> newBranch -> originalBranch
	if err := eng.SetParent(ctx, branchToSplit, newBranchName); err != nil {
		return nil, fmt.Errorf("failed to update parent: %w", err)
	}

	return &SplitResult{
		BranchNames:  []string{newBranchName},
		BranchPoints: []int{0}, // Single commit
	}, nil
}

// Helper functions

func promptBranchName(ctx context.Context, existingNames []string, originalBranchName string, branchNum int, eng engine.BranchReader) (string, error) {
	defaultName := originalBranchName
	if containsString(existingNames, defaultName) {
		defaultName = originalBranchName + "_split"
		for containsString(existingNames, defaultName) {
			defaultName = defaultName + "_split"
		}
	}

	var branchName string
	prompt := &survey.Input{
		Message: fmt.Sprintf("Choose a name for branch %d", branchNum),
		Default: defaultName,
	}
	if err := survey.AskOne(prompt, &branchName); err != nil {
		return "", fmt.Errorf("canceled")
	}

	// Validate name - don't allow names already picked in this split session
	if containsString(existingNames, branchName) {
		return "", fmt.Errorf("branch name %s is already used by another branch in this split", branchName)
	}

	// Allow reusing the original branch name being split (it will be replaced)
	// but don't allow other existing branch names
	if branchName != originalBranchName {
		allBranches := eng.AllBranchNames()
		if containsString(allBranches, branchName) {
			return "", fmt.Errorf("branch name %s is already in use", branchName)
		}
	}

	return branchName, nil
}

// promptForFiles shows an interactive file selector for split --by-file
func promptForFiles(ctx context.Context, branchToSplit string, eng engine.Engine, splog *tui.Splog) ([]string, error) {
	// Get the parent branch to compare against
	parentBranchName := eng.GetParentPrecondition(branchToSplit)

	// Get merge base between branch and parent
	mergeBase, err := git.GetMergeBase(ctx, branchToSplit, parentBranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get list of changed files
	changedFiles, err := git.GetChangedFiles(ctx, mergeBase, branchToSplit)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		return nil, fmt.Errorf("no files changed in branch %s", branchToSplit)
	}

	if len(changedFiles) == 1 {
		return nil, fmt.Errorf("only one file changed in branch - nothing to split")
	}

	// Show instructions
	splog.Info("Splitting %s by file.", tui.ColorBranchName(branchToSplit, true))
	splog.Info("Select the files to extract to a new parent branch.")
	splog.Info("The remaining files will stay on %s.", tui.ColorBranchName(branchToSplit, true))
	splog.Info("")

	// Prompt for file selection
	var selectedFiles []string
	prompt := &survey.MultiSelect{
		Message: "Select files to extract:",
		Options: changedFiles,
	}
	if err := survey.AskOne(prompt, &selectedFiles); err != nil {
		return nil, fmt.Errorf("canceled")
	}

	// Validate that not all files were selected
	if len(selectedFiles) == len(changedFiles) {
		return nil, fmt.Errorf("cannot extract all files - at least one must remain on the original branch")
	}

	return selectedFiles, nil
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "ren" // "child" -> "children"
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func makeRange(n int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i
	}
	return result
}

func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("GIT_EDITOR")
	}
	if editor == "" {
		// Try to get from git config
		output, err := exec.Command("git", "config", "--get", "core.editor").Output()
		if err == nil {
			editor = strings.TrimSpace(string(output))
		}
	}
	if editor == "" {
		editor = "vi" // Default fallback
	}
	return editor
}
