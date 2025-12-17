package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// CheckoutOptions specifies options for the checkout command
type CheckoutOptions struct {
	BranchName    string // Optional: branch to checkout directly
	ShowUntracked bool   // Include untracked branches in selection
	All           bool   // Show all branches across trunks
	StackOnly     bool   // Only show current stack (ancestors + descendants)
	CheckoutTrunk bool   // Checkout trunk directly
}

// CheckoutAction performs the checkout operation
func CheckoutAction(opts CheckoutOptions, ctx *runtime.Context) error {
	// Populate remote SHAs if needed
	if err := ctx.Engine.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	var branchName string
	var err error

	// Handle --trunk flag
	if opts.CheckoutTrunk {
		branchName = ctx.Engine.Trunk()
	} else if opts.BranchName != "" {
		// Direct checkout
		branchName = opts.BranchName
	} else {
		// Interactive selection
		branchName, err = interactiveBranchSelection(opts, ctx)
		if err != nil {
			return err
		}
	}

	// Check if already on the branch
	currentBranch := ctx.Engine.CurrentBranch()
	if branchName == currentBranch {
		ctx.Splog.Info("Already on %s.", output.ColorBranchName(branchName, true))
		return nil
	}

	// Checkout the branch
	if err := git.CheckoutBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	ctx.Splog.Info("Checked out %s.", output.ColorBranchName(branchName, false))
	printBranchInfo(branchName, ctx)

	return nil
}

// interactiveBranchSelection shows an interactive branch selector
func interactiveBranchSelection(opts CheckoutOptions, ctx *runtime.Context) (string, error) {
	var choices []branchChoice
	var initialIndex int = -1
	currentBranch := ctx.Engine.CurrentBranch()
	seenBranches := make(map[string]bool)

	if opts.StackOnly {
		// Only show current stack (ancestors + descendants)
		if currentBranch == "" {
			return "", fmt.Errorf("not on a branch; cannot use --stack flag")
		}

		scope := engine.Scope{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := ctx.Engine.GetRelativeStack(currentBranch, scope)

		// Build choices from stack
		for _, branchName := range stack {
			if seenBranches[branchName] {
				continue
			}
			seenBranches[branchName] = true
			display := branchName
			if branchName == currentBranch {
				display = output.ColorBranchName(branchName, true)
				initialIndex = len(choices)
			} else {
				display = output.ColorBranchName(branchName, false)
			}
			choices = append(choices, branchChoice{
				display: display,
				value:   branchName,
			})
		}
	} else {
		// Get branches in stack order: trunk first, then children recursively
		trunkName := ctx.Engine.Trunk()
		branchOrder := collectBranchesDepthFirst(trunkName, ctx)

		for _, branchName := range branchOrder {
			if seenBranches[branchName] {
				continue
			}
			seenBranches[branchName] = true

			isCurrent := branchName == currentBranch
			display := output.ColorBranchName(branchName, isCurrent)
			if isCurrent {
				initialIndex = len(choices)
			}
			choices = append(choices, branchChoice{
				display: display,
				value:   branchName,
			})
		}
	}

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchNames(ctx)
		for _, branchName := range untracked {
			if !seenBranches[branchName] {
				choices = append(choices, branchChoice{
					display: branchName,
					value:   branchName,
				})
				seenBranches[branchName] = true
			}
		}
	}

	// Fallback: if we still have no choices, get all branches directly from engine
	if len(choices) == 0 {
		allBranches := ctx.Engine.AllBranchNames()
		trunkName := ctx.Engine.Trunk()

		// Ensure trunk is always included
		if trunkName != "" && !seenBranches[trunkName] {
			display := trunkName
			if trunkName == currentBranch {
				display = output.ColorBranchName(trunkName, true)
				initialIndex = 0
			} else {
				display = output.ColorBranchName(trunkName, false)
			}
			choices = append(choices, branchChoice{
				display: display,
				value:   trunkName,
			})
			seenBranches[trunkName] = true
		}

		// Add all other branches
		for _, branchName := range allBranches {
			if !seenBranches[branchName] {
				display := branchName
				if branchName == currentBranch {
					display = output.ColorBranchName(branchName, true)
					initialIndex = len(choices)
				} else {
					display = output.ColorBranchName(branchName, false)
				}
				choices = append(choices, branchChoice{
					display: display,
					value:   branchName,
				})
				seenBranches[branchName] = true
			}
		}

		if len(choices) == 0 {
			return "", fmt.Errorf("no branches available to checkout")
		}
	}

	if len(choices) == 0 {
		return "", fmt.Errorf("no branches available to checkout")
	}

	// Set initial index if not found
	if initialIndex < 0 {
		initialIndex = len(choices) - 1
	}

	// Show interactive selector
	selected, err := promptBranchSelection("Checkout a branch (arrow keys to navigate, type to filter)", choices, initialIndex)
	if err != nil {
		return "", err
	}

	return selected, nil
}

// printBranchInfo prints information about the checked out branch
func printBranchInfo(branchName string, ctx *runtime.Context) {
	if ctx.Engine.IsTrunk(branchName) {
		return
	}

	if !ctx.Engine.IsBranchTracked(branchName) {
		ctx.Splog.Info("This branch is not tracked by Stackit.")
		return
	}

	if !ctx.Engine.IsBranchFixed(branchName) {
		parent := ctx.Engine.GetParentPrecondition(branchName)
		ctx.Splog.Info("This branch has fallen behind %s - you may want to %s.",
			output.ColorBranchName(parent, false),
			output.ColorCyan("stackit upstack restack"))
		return
	}

	// Check if any downstack branch needs restack
	scope := engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	downstack := ctx.Engine.GetRelativeStack(branchName, scope)

	// Reverse to check from trunk upward
	for i := len(downstack) - 1; i >= 0; i-- {
		ancestor := downstack[i]
		if !ctx.Engine.IsBranchFixed(ancestor) {
			parent := ctx.Engine.GetParentPrecondition(ancestor)
			ctx.Splog.Info("The downstack branch %s has fallen behind %s - you may want to %s.",
				output.ColorBranchName(ancestor, false),
				output.ColorBranchName(parent, false),
				output.ColorCyan("stackit stack restack"))
			return
		}
	}
}

// collectBranchesDepthFirst returns branches with trunk first, then children recursively
func collectBranchesDepthFirst(branchName string, ctx *runtime.Context) []string {
	var result []string
	result = append(result, branchName)

	children := ctx.Engine.GetChildren(branchName)
	for _, child := range children {
		result = append(result, collectBranchesDepthFirst(child, ctx)...)
	}

	return result
}
