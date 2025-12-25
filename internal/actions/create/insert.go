package create

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

func handleInsert(ctx context.Context, newBranch, currentBranch string, runtimeCtx *runtime.Context, opts *Options) error {
	currentBranchObj := runtimeCtx.Engine.GetBranch(currentBranch)
	children := currentBranchObj.GetChildren()
	siblings := []string{}
	for _, child := range children {
		if child.Name != newBranch {
			siblings = append(siblings, child.Name)
		}
	}

	if len(siblings) == 0 {
		return nil
	}

	// If multiple children, prompt user to select which to move
	var toMove []string
	switch {
	case len(opts.SelectedChildren) > 0:
		// Use pre-selected children (for tests)
		for _, selected := range opts.SelectedChildren {
			for _, sibling := range siblings {
				if selected == sibling {
					toMove = append(toMove, sibling)
					break
				}
			}
		}
	case len(siblings) > 1 && utils.IsInteractive():
		runtimeCtx.Splog.Info("Current branch has multiple children. Select which should be moved onto the new branch:")
		options := []tui.SelectOption{
			{Label: "All children", Value: "all"},
		}
		for _, child := range siblings {
			options = append(options, tui.SelectOption{Label: child, Value: child})
		}

		selected, err := tui.PromptSelect("Which child should be moved onto the new branch?", options, 0)
		if err != nil {
			return err
		}

		if selected == "all" {
			toMove = siblings
		} else {
			toMove = []string{selected}
		}
	default:
		// Single child or non-interactive - move all
		toMove = siblings
	}

	// Update parent for each child to move
	branchesToRestack := make([]engine.Branch, 0, len(toMove))
	for _, child := range toMove {
		if err := runtimeCtx.Engine.TrackBranch(ctx, child, newBranch); err != nil {
			return fmt.Errorf("failed to update parent for %s: %w", child, err)
		}
		branchesToRestack = append(branchesToRestack, runtimeCtx.Engine.GetBranch(child))
	}

	// Restack children onto the new branch to physically insert it
	if len(branchesToRestack) > 0 {
		batchRes, err := runtimeCtx.Engine.RestackBranches(ctx, branchesToRestack)
		if err != nil {
			runtimeCtx.Splog.Info("Warning: failed to restack branches onto %s: %v", newBranch, err)
		}

		for _, child := range toMove {
			res, ok := batchRes.Results[child]
			if !ok {
				continue
			}

			if res.Result == engine.RestackConflict {
				runtimeCtx.Splog.Info("Conflict restacking %s onto %s. Please resolve manually or run 'stackit sync --restack'.", child, newBranch)
			} else if res.Result == engine.RestackDone {
				runtimeCtx.Splog.Info("Restacked %s onto %s.", child, newBranch)
			}
		}
	}

	return nil
}
