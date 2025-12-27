package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// UntrackOptions contains options for the untrack command
type UntrackOptions struct {
	BranchName string
	Force      bool
}

// UntrackAction performs the untrack operation
func UntrackAction(ctx *runtime.Context, opts UntrackOptions) error {
	eng := ctx.Engine
	branchName := opts.BranchName

	// Check if branch is tracked
	branch := eng.GetBranch(branchName)
	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked", branchName)
	}

	// Find descendants
	descendants := eng.GetRelativeStackUpstack(branch)

	// If there are descendants and not forced, prompt for confirmation
	if len(descendants) > 0 && !opts.Force {
		message := fmt.Sprintf("Branch %s has %d tracked descendants. Untrack all of them?",
			style.ColorBranchName(branchName, false), len(descendants))
		options := []tui.SelectOption{
			{Label: "Yes", Value: yesResponse},
			{Label: "No", Value: noResponse},
		}

		selected, err := tui.PromptSelect(message, options, 0)
		if err != nil {
			return err
		}

		if selected != yesResponse {
			ctx.Splog.Info("Untrack canceled.")
			return nil
		}
	}

	// Untrack recursively (descendants first, then the branch itself)
	// Actually order doesn't strictly matter for metadata deletion but it's cleaner
	for _, descendant := range descendants {
		if err := eng.UntrackBranch(descendant.Name); err != nil {
			return fmt.Errorf("failed to untrack descendant %s: %w", descendant.Name, err)
		}
		ctx.Splog.Info("Stopped tracking %s.", style.ColorBranchName(descendant.Name, false))
	}

	if err := eng.UntrackBranch(branchName); err != nil {
		return fmt.Errorf("failed to untrack branch %s: %w", branchName, err)
	}
	ctx.Splog.Info("Stopped tracking %s.", style.ColorBranchName(branchName, false))

	return nil
}
