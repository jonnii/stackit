package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
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
	if !eng.IsBranchTracked(branchName) {
		return fmt.Errorf("branch %s is not tracked", branchName)
	}

	// Find descendants
	descendants := eng.GetRelativeStackUpstack(branchName)

	// If there are descendants and not forced, prompt for confirmation
	if len(descendants) > 0 && !opts.Force {
		message := fmt.Sprintf("Branch %s has %d tracked descendants. Untrack all of them?",
			tui.ColorBranchName(branchName, false), len(descendants))
		options := []tui.SelectOption{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}

		selected, err := tui.PromptSelect(message, options, 0)
		if err != nil {
			return err
		}

		if selected != "yes" {
			ctx.Splog.Info("Untrack canceled.")
			return nil
		}
	}

	// Untrack recursively (descendants first, then the branch itself)
	// Actually order doesn't strictly matter for metadata deletion but it's cleaner
	for _, descendant := range descendants {
		if err := eng.UntrackBranch(ctx.Context, descendant); err != nil {
			return fmt.Errorf("failed to untrack descendant %s: %w", descendant, err)
		}
		ctx.Splog.Info("Stopped tracking %s.", tui.ColorBranchName(descendant, false))
	}

	if err := eng.UntrackBranch(ctx.Context, branchName); err != nil {
		return fmt.Errorf("failed to untrack branch %s: %w", branchName, err)
	}
	ctx.Splog.Info("Stopped tracking %s.", tui.ColorBranchName(branchName, false))

	return nil
}
