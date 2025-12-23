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
	if err := eng.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	var branchName string

	// Handle --trunk flag
	switch {
	case opts.CheckoutTrunk:
		branchName = eng.Trunk().Name
	case opts.BranchName != "":
		// Direct checkout
		branchName = opts.BranchName
	default:
		// Interactive selection
		branches, err := buildBranchChoices(ctx, opts)
		if err != nil {
			return err
		}
		// Convert []Branch to []string for TUI
		branchNames := make([]string, len(branches))
		for i, branch := range branches {
			branchNames[i] = branch.Name
		}
		currentBranch := eng.CurrentBranch()
		branchName, err = tui.PromptBranchCheckout(branchNames, currentBranch.Name)
		if err != nil {
			return err
		}
	}

	// Check if already on the branch
	currentBranch := eng.CurrentBranch()
	if branchName == currentBranch.Name {
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

// getUntrackedBranchesForCheckout returns all untracked branches (excluding trunk)
func getUntrackedBranchesForCheckout(eng engine.BranchReader) []engine.Branch {
	var untracked []engine.Branch
	for _, branch := range eng.AllBranches() {
		if !branch.IsTrunk() && !branch.IsTracked() {
			untracked = append(untracked, branch)
		}
	}
	return untracked
}

// buildBranchChoices builds the list of branches to show in the interactive selector
func buildBranchChoices(ctx *runtime.Context, opts CheckoutOptions) ([]engine.Branch, error) {
	eng := ctx.Engine
	currentBranch := eng.CurrentBranch()
	trunk := eng.Trunk()
	seenBranches := make(map[string]bool)
	var branches []engine.Branch

	if opts.StackOnly {
		// Only show current stack (ancestors + descendants)
		if currentBranch == nil {
			return nil, fmt.Errorf("not on a branch; cannot use --stack flag")
		}

		scope := engine.Scope{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := eng.GetRelativeStack(currentBranch.Name, scope)

		// Build branch list from stack
		for _, branch := range stack {
			if seenBranches[branch.Name] {
				continue
			}
			seenBranches[branch.Name] = true
			branches = append(branches, branch)
		}
	} else {
		// Get branches in stack order: trunk first, then children recursively
		for branch := range eng.BranchesDepthFirst(trunk.Name) {
			if seenBranches[branch.Name] {
				continue
			}
			seenBranches[branch.Name] = true
			branches = append(branches, branch)
		}
	}

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchesForCheckout(eng)
		for _, branch := range untracked {
			if !seenBranches[branch.Name] {
				branches = append(branches, branch)
				seenBranches[branch.Name] = true
			}
		}
	}

	// Fallback: if we still have no choices, get all branches directly from engine
	if len(branches) == 0 {
		allBranches := eng.AllBranches()

		// Ensure trunk is always included
		if trunk.Name != "" && !seenBranches[trunk.Name] {
			branches = append(branches, trunk)
			seenBranches[trunk.Name] = true
		}

		// Add all other branches
		for _, branch := range allBranches {
			if !seenBranches[branch.Name] {
				branches = append(branches, branch)
				seenBranches[branch.Name] = true
			}
		}

		if len(branches) == 0 {
			return nil, fmt.Errorf("no branches available to checkout")
		}
	}

	return branches, nil
}

// printBranchInfo prints information about the checked out branch
func printBranchInfo(branchName string, ctx *runtime.Context) {
	branch := ctx.Engine.GetBranch(branchName)
	if branch.IsTrunk() {
		return
	}

	if !branch.IsTracked() {
		ctx.Splog.Info("This branch is not tracked by Stackit.")
		return
	}

	if !ctx.Engine.IsBranchUpToDate(branchName) {
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
		if !ctx.Engine.IsBranchUpToDate(ancestor.Name) {
			parent := ctx.Engine.GetParentPrecondition(ancestor.Name)
			ctx.Splog.Info("The downstack branch %s has fallen behind %s - you may want to %s.",
				tui.ColorBranchName(ancestor.Name, false),
				tui.ColorBranchName(parent, false),
				tui.ColorCyan("stackit stack restack"))
			return
		}
	}
}
