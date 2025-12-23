package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// newMoveCmd creates the move command
func newMoveCmd() *cobra.Command {
	var (
		all    bool
		onto   string
		source string
	)

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Rebase the current branch onto the target branch",
		Long: `Rebase the current branch onto the target branch and restack all of its descendants.

If no branch is passed in, opens an interactive selector to choose the target branch.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Default source to current branch
			sourceBranch := source
			if sourceBranch == "" {
				currentBranch := ctx.Engine.CurrentBranch()
				if currentBranch == nil {
					return fmt.Errorf("not on a branch and no source branch specified")
				}
				sourceBranch = currentBranch.Name
			}

			// Handle interactive selection for onto if not provided
			ontoBranch := onto
			if ontoBranch == "" {
				var err error
				ontoBranch, err = interactiveOntoSelection(ctx, sourceBranch)
				if err != nil {
					return err
				}
			}

			// Run move action
			return actions.MoveAction(ctx, actions.MoveOptions{
				Source: sourceBranch,
				Onto:   ontoBranch,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show branches across all configured trunks in interactive selection.")
	cmd.Flags().StringVarP(&onto, "onto", "o", "", "Branch to move the current branch onto.")
	cmd.Flags().StringVar(&source, "source", "", "Branch to move (defaults to current branch).")

	_ = cmd.RegisterFlagCompletionFunc("onto", completeBranches)
	_ = cmd.RegisterFlagCompletionFunc("source", completeBranches)

	return cmd
}

// interactiveOntoSelection shows an interactive branch selector for choosing the "onto" branch
func interactiveOntoSelection(ctx *runtime.Context, sourceBranch string) (string, error) {
	eng := ctx.Engine
	initialIndex := -1
	seenBranches := make(map[string]bool)

	// Get descendants of source to exclude them
	sourceBranchObj := eng.GetBranch(sourceBranch)
	descendants := sourceBranchObj.GetRelativeStack(engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})
	excludedBranches := make(map[string]bool)
	for _, d := range descendants {
		excludedBranches[d.Name] = true
	}

	// Get branches in stack order: trunk first, then children recursively
	trunk := eng.Trunk()
	choices := make([]tui.BranchChoice, 0)

	for branch := range eng.BranchesDepthFirst(trunk) {
		// Skip source and its descendants
		if excludedBranches[branch.Name] {
			continue
		}

		if seenBranches[branch.Name] {
			continue
		}
		seenBranches[branch.Name] = true

		currentBranch := eng.CurrentBranch()
		isCurrent := branch.Name == currentBranch.Name
		display := tui.ColorBranchName(branch.Name, isCurrent)
		if isCurrent {
			initialIndex = len(choices)
		}
		choices = append(choices, tui.BranchChoice{
			Display: display,
			Value:   branch.Name,
		})
	}

	// Fallback: if we still have no choices, get all branches directly from engine
	if len(choices) == 0 {
		allBranches := eng.AllBranches()

		// Ensure trunk is always included if not excluded
		if trunk.Name != "" && !excludedBranches[trunk.Name] && !seenBranches[trunk.Name] {
			currentBranch := eng.CurrentBranch()
			var display string
			if trunk.Name == currentBranch.Name {
				display = tui.ColorBranchName(trunk.Name, true)
				initialIndex = 0
			} else {
				display = tui.ColorBranchName(trunk.Name, false)
			}
			choices = append(choices, tui.BranchChoice{
				Display: display,
				Value:   trunk.Name,
			})
			seenBranches[trunk.Name] = true
		}

		// Add all other branches
		for _, branch := range allBranches {
			branchName := branch.Name
			if excludedBranches[branchName] {
				continue
			}
			if !seenBranches[branchName] {
				var display string
				currentBranch := eng.CurrentBranch()
				if branchName == currentBranch.Name {
					display = tui.ColorBranchName(branchName, true)
					initialIndex = len(choices)
				} else {
					display = tui.ColorBranchName(branchName, false)
				}
				choices = append(choices, tui.BranchChoice{
					Display: display,
					Value:   branchName,
				})
				seenBranches[branchName] = true
			}
		}

		if len(choices) == 0 {
			return "", fmt.Errorf("no valid branches available to move onto")
		}
	}

	if len(choices) == 0 {
		return "", fmt.Errorf("no valid branches available to move onto")
	}

	// Set initial index if not found
	if initialIndex < 0 {
		initialIndex = len(choices) - 1
	}

	// Show interactive selector
	selected, err := tui.PromptBranchSelection("Move branch onto (arrow keys to navigate, type to filter)", choices, initialIndex)
	if err != nil {
		return "", err
	}

	return selected, nil
}
