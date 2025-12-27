package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
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

	if err := eng.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	var branchName string

	switch {
	case opts.CheckoutTrunk:
		branchName = eng.Trunk().GetName()
	case opts.BranchName != "":
		branchName = opts.BranchName
	default:
		if !utils.IsInteractive() {
			return fmt.Errorf("interactive branch selection is not available in non-interactive mode; please specify a branch name")
		}
		branches, err := buildBranchChoices(ctx, opts)
		if err != nil {
			return err
		}
		branchName, err = tui.PromptBranchCheckout(branches, eng)
		if err != nil {
			return err
		}
	}

	currentBranch := eng.CurrentBranch()
	if currentBranch != nil && branchName == currentBranch.GetName() {
		splog.Info("Already on %s.", style.ColorBranchName(branchName, true))
		return nil
	}

	branch := eng.GetBranch(branchName)
	if err := eng.CheckoutBranch(context, branch); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	splog.Info("Checked out %s.", style.ColorBranchName(branchName, false))
	printBranchInfo(ctx, branch)

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
		if currentBranch == nil {
			return nil, fmt.Errorf("not on a branch; cannot use --stack flag")
		}

		rng := engine.StackRange{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := currentBranch.GetRelativeStack(rng)

		for _, branch := range stack {
			if seenBranches[branch.GetName()] {
				continue
			}
			seenBranches[branch.GetName()] = true
			branches = append(branches, branch)
		}
	} else {
		// Get branches in stack order: trunk first, then children recursively
		for branch := range eng.BranchesDepthFirst(trunk) {
			if seenBranches[branch.GetName()] {
				continue
			}
			seenBranches[branch.GetName()] = true
			branches = append(branches, branch)
		}
	}

	if opts.ShowUntracked {
		untracked := getUntrackedBranchesForCheckout(eng)
		for _, branch := range untracked {
			if !seenBranches[branch.GetName()] {
				branches = append(branches, branch)
				seenBranches[branch.GetName()] = true
			}
		}
	}

	// Fallback: get all branches if none found yet
	if len(branches) == 0 {
		allBranches := eng.AllBranches()

		if trunk.GetName() != "" && !seenBranches[trunk.GetName()] {
			branches = append(branches, trunk)
			seenBranches[trunk.GetName()] = true
		}

		for _, branch := range allBranches {
			if !seenBranches[branch.GetName()] {
				branches = append(branches, branch)
				seenBranches[branch.GetName()] = true
			}
		}

		if len(branches) == 0 {
			return nil, fmt.Errorf("no branches available to checkout")
		}
	}

	return branches, nil
}

func printBranchInfo(ctx *runtime.Context, branch engine.Branch) {
	if branch.IsTrunk() {
		return
	}

	if !branch.IsTracked() {
		ctx.Splog.Info("This branch is not tracked by Stackit.")
		return
	}

	if !branch.IsBranchUpToDate() {
		parent := branch.GetParentPrecondition()
		ctx.Splog.Info("This branch has fallen behind %s - you may want to %s.",
			style.ColorBranchName(parent, false),
			style.ColorCyan("stackit upstack restack"))
		return
	}

	// Check if any downstack branch needs restack
	rng := engine.StackRange{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	downstack := branch.GetRelativeStack(rng)

	// Check from trunk upward
	for i := len(downstack) - 1; i >= 0; i-- {
		ancestor := downstack[i]
		if !ancestor.IsBranchUpToDate() {
			parent := ancestor.GetParentPrecondition()
			ctx.Splog.Info("The downstack branch %s has fallen behind %s - you may want to %s.",
				style.ColorBranchName(ancestor.GetName(), false),
				style.ColorBranchName(parent, false),
				style.ColorCyan("stackit stack restack"))
			return
		}
	}
}
