package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// ScopeOptions contains options for the scope command
type ScopeOptions struct {
	Scope string
	Unset bool
	Show  bool
}

// ScopeAction implements the stackit scope command
func ScopeAction(ctx *runtime.Context, opts ScopeOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch, _ := git.GetCurrentBranch()
	isOnTrunk := currentBranch == eng.Trunk().Name || currentBranch == ""

	// Handle Show
	if opts.Show {
		if isOnTrunk {
			return fmt.Errorf("not on a branch")
		}
		explicitScope := eng.GetExplicitScopeInternal(currentBranch)
		resolvedScope := eng.GetScopeInternal(currentBranch)

		switch {
		case !explicitScope.IsEmpty():
			if explicitScope.IsNone() {
				splog.Info("Branch %s has scope inheritance DISABLED (explicitly set to '%s').", tui.ColorBranchName(currentBranch, false), explicitScope.String())
			} else {
				splog.Info("Branch %s has explicit scope: %s", tui.ColorBranchName(currentBranch, false), tui.ColorDim(explicitScope.String()))
			}
		case !resolvedScope.IsEmpty():
			splog.Info("Branch %s inherits scope: %s", tui.ColorBranchName(currentBranch, false), tui.ColorDim(resolvedScope.String()))
		default:
			splog.Info("Branch %s has no scope set.", tui.ColorBranchName(currentBranch, false))
		}
		return nil
	}

	// Handle Unset
	if opts.Unset {
		if isOnTrunk {
			return fmt.Errorf("cannot unset scope on trunk")
		}
		if err := eng.SetScope(eng.GetBranch(currentBranch), engine.Empty()); err != nil {
			return fmt.Errorf("failed to unset scope: %w", err)
		}
		splog.Info("Unset explicit scope for branch %s. It will now inherit from its parent.", tui.ColorBranchName(currentBranch, false))
		return nil
	}

	// If no scope provided and not show/unset, we don't know what to do
	if opts.Scope == "" {
		return fmt.Errorf("no scope name provided")
	}

	// Update the current branch's scope
	oldScope := eng.GetScopeInternal(currentBranch)
	newScope := engine.NewScope(opts.Scope)
	if err := eng.SetScope(eng.GetBranch(currentBranch), newScope); err != nil {
		return fmt.Errorf("failed to set scope: %w", err)
	}

	if newScope.IsNone() {
		splog.Info("Disabled scope for branch %s (breaks inheritance).", tui.ColorBranchName(currentBranch, false))
	} else {
		splog.Info("Set scope for branch %s to: %s", tui.ColorBranchName(currentBranch, false), tui.ColorDim(opts.Scope))

		// Rename prompt
		if oldScope.IsDefined() && !oldScope.Equal(newScope) && utils.IsInteractive() && strings.Contains(currentBranch, oldScope.String()) {
			confirmed, err := tui.PromptConfirm(fmt.Sprintf("Branch name contains '%s', but its scope is now '%s'. Would you like to rename the branch?", oldScope.String(), opts.Scope), true)
			if err == nil && confirmed {
				newName := strings.Replace(currentBranch, oldScope.String(), opts.Scope, 1)
				if err := eng.RenameBranch(ctx.Context, eng.GetBranch(currentBranch), eng.GetBranch(newName)); err != nil {
					splog.Info("Warning: failed to rename branch: %v", err)
				} else {
					splog.Info("Renamed branch %s to %s.", tui.ColorBranchName(currentBranch, false), tui.ColorBranchName(newName, true))
				}
			}
		}
	}

	return nil
}
