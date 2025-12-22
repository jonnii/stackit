package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// CheckoutOptions contains options for the checkout command
type CheckoutOptions struct {
	BranchName    string // Optional: branch to checkout directly
	ShowUntracked bool   // Include untracked branches in selection
	All           bool   // Show all branches across trunks
	StackOnly     bool   // Only show current stack (ancestors + descendants)
	CheckoutTrunk bool   // Checkout trunk directly
}

// CheckoutAction performs the checkout operation
func CheckoutAction(ctx *runtime.Context, opts CheckoutOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	context := ctx.Context

	// Populate remote SHAs if needed
	if err := eng.PopulateRemoteShas(context); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	var branchName string

	// Handle --trunk flag
	switch {
	case opts.CheckoutTrunk:
		branchName = eng.Trunk()
	case opts.BranchName != "":
		// Direct checkout
		branchName = opts.BranchName
	default:
		// Interactive selection
		branchNames, err := buildBranchChoices(ctx, opts)
		if err != nil {
			return err
		}
		branchName, err = tui.PromptBranchCheckout(branchNames, eng.CurrentBranch())
		if err != nil {
			return err
		}
	}

	// Check if already on the branch
	currentBranch := eng.CurrentBranch()
	if branchName == currentBranch {
		splog.Info("Already on %s.", tui.ColorBranchName(branchName, true))
		return nil
	}

	// Checkout the branch
	if err := git.CheckoutBranch(context, branchName); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	splog.Info("Checked out %s.", tui.ColorBranchName(branchName, false))
	printBranchInfo(branchName, ctx)

	return nil
}

// collectBranchesDepthFirst returns branches with trunk first, then children recursively
func collectBranchesDepthFirst(branchName string, eng engine.BranchReader) []string {
	var result []string
	result = append(result, branchName)

	children := eng.GetChildren(branchName)
	for _, child := range children {
		result = append(result, collectBranchesDepthFirst(child, eng)...)
	}

	return result
}

// getUntrackedBranchNamesForCheckout returns all untracked branch names (excluding trunk)
func getUntrackedBranchNamesForCheckout(eng engine.BranchReader) []string {
	var untracked []string
	for _, branchName := range eng.AllBranchNames() {
		if !eng.IsTrunk(branchName) && !eng.IsBranchTracked(branchName) {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}

// buildBranchChoices builds the list of branch names to show in the interactive selector
func buildBranchChoices(ctx *runtime.Context, opts CheckoutOptions) ([]string, error) {
	eng := ctx.Engine
	currentBranch := eng.CurrentBranch()
	seenBranches := make(map[string]bool)
	var branchNames []string

	if opts.StackOnly {
		// Only show current stack (ancestors + descendants)
		if currentBranch == "" {
			return nil, fmt.Errorf("not on a branch; cannot use --stack flag")
		}

		scope := engine.Scope{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := eng.GetRelativeStack(currentBranch, scope)

		// Build branch list from stack
		for _, branchName := range stack {
			if seenBranches[branchName] {
				continue
			}
			seenBranches[branchName] = true
			branchNames = append(branchNames, branchName)
		}
	} else {
		// Get branches in stack order: trunk first, then children recursively
		trunkName := eng.Trunk()
		branchOrder := collectBranchesDepthFirst(trunkName, eng)

		for _, branchName := range branchOrder {
			if seenBranches[branchName] {
				continue
			}
			seenBranches[branchName] = true
			branchNames = append(branchNames, branchName)
		}
	}

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchNamesForCheckout(eng)
		for _, branchName := range untracked {
			if !seenBranches[branchName] {
				branchNames = append(branchNames, branchName)
				seenBranches[branchName] = true
			}
		}
	}

	// Fallback: if we still have no choices, get all branches directly from engine
	if len(branchNames) == 0 {
		allBranches := eng.AllBranchNames()
		trunkName := eng.Trunk()

		// Ensure trunk is always included
		if trunkName != "" && !seenBranches[trunkName] {
			branchNames = append(branchNames, trunkName)
			seenBranches[trunkName] = true
		}

		// Add all other branches
		for _, branchName := range allBranches {
			if !seenBranches[branchName] {
				branchNames = append(branchNames, branchName)
				seenBranches[branchName] = true
			}
		}

		if len(branchNames) == 0 {
			return nil, fmt.Errorf("no branches available to checkout")
		}
	}

	return branchNames, nil
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

	if !ctx.Engine.IsBranchFixed(ctx.Context, branchName) {
		parent := ctx.Engine.GetParentPrecondition(branchName)
		ctx.Splog.Info("This branch has fallen behind %s - you may want to %s.",
			tui.ColorBranchName(parent, false),
			tui.ColorCyan("stackit upstack restack"))
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
		if !ctx.Engine.IsBranchFixed(ctx.Context, ancestor) {
			parent := ctx.Engine.GetParentPrecondition(ancestor)
			ctx.Splog.Info("The downstack branch %s has fallen behind %s - you may want to %s.",
				tui.ColorBranchName(ancestor, false),
				tui.ColorBranchName(parent, false),
				tui.ColorCyan("stackit stack restack"))
			return
		}
	}
}
