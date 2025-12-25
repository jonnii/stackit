package fold

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the fold command
type Options struct {
	Keep       bool // If true, keeps the name of the current branch instead of using the name of its parent
	AllowTrunk bool // If true, allows folding into the trunk branch
}

// Action performs the fold operation
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Validate we're on a branch
	currentBranch, err := utils.ValidateOnBranch(ctx.Engine)
	if err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	snapshotOpts := actions.NewSnapshot("fold",
		actions.WithFlag(opts.Keep, "--keep"),
		actions.WithFlag(opts.AllowTrunk, "--allow-trunk"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Check if on trunk
	currentBranchObj := eng.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return fmt.Errorf("cannot fold trunk branch")
	}

	// Check if branch is tracked
	if !currentBranchObj.IsTracked() {
		return fmt.Errorf("cannot fold untracked branch %s", currentBranch)
	}

	// Check if rebase is in progress
	if err := utils.CheckRebaseInProgress(gctx); err != nil {
		return err
	}

	// Check for uncommitted changes
	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("cannot fold with uncommitted changes. Please commit or stash them first")
	}

	// Get parent branch
	// currentBranchObj already declared above
	parent := eng.GetParent(currentBranchObj)
	parentName := ""
	if parent == nil {
		parentName = eng.Trunk().Name
	} else {
		parentName = parent.Name
	}

	parentBranch := eng.GetBranch(parentName)

	// Prohibit folding branches with different scopes
	if !parentBranch.IsTrunk() {
		currentScope := currentBranchObj.GetScope()
		parentScope := parentBranch.GetScope()
		if !currentScope.Equal(parentScope) {
			return fmt.Errorf("cannot fold branches with different scopes (current: [%s], parent: [%s])", currentScope.String(), parentScope.String())
		}
	}

	if opts.Keep {
		// Prevent folding onto trunk with --keep, as that would delete trunk
		if parentBranch.IsTrunk() {
			return fmt.Errorf("cannot fold into trunk with --keep because it would delete the trunk branch")
		}
		return foldWithKeep(gctx, ctx, currentBranch, parentName, eng, splog, opts)
	}

	// Check if folding into trunk
	if parentBranch.IsTrunk() && !opts.AllowTrunk {
		return fmt.Errorf("cannot fold into trunk branch %s without --allow-trunk. Folding into trunk will modify your local main branch directly", parentName)
	}

	return foldNormal(gctx, ctx, currentBranch, parentName, eng, splog, opts)
}
