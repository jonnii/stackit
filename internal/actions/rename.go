package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// RenameOptions contains options for the rename command
type RenameOptions struct {
	NewName string
	Force   bool
}

// RenameAction renames the current branch and updates metadata
func RenameAction(ctx *runtime.Context, opts RenameOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch, err := utils.ValidateOnBranch(ctx.Engine)
	if err != nil {
		return err
	}

	// Validate not renaming trunk
	if currentBranch == eng.Trunk().Name {
		return fmt.Errorf("cannot rename trunk branch %s", currentBranch)
	}

	// Determine new name
	newName := opts.NewName
	if newName == "" {
		if !utils.IsInteractive() {
			return fmt.Errorf("branch name is required in non-interactive mode")
		}

		newName, err = tui.PromptTextInput("Enter new branch name:", currentBranch)
		if err != nil {
			return err
		}
	}

	// Sanitize new name
	newName = utils.SanitizeBranchName(newName)
	if newName == "" {
		return fmt.Errorf("invalid branch name")
	}

	if newName == currentBranch {
		splog.Info("Branch is already named %s.", newName)
		return nil
	}

	// Check if new name already exists
	allBranches, err := git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to check existing branches: %w", err)
	}
	for _, b := range allBranches {
		if b == newName {
			return fmt.Errorf("branch %s already exists", newName)
		}
	}

	// Check for PR association
	meta, err := git.ReadMetadataRef(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if meta.PrInfo != nil && meta.PrInfo.Number != nil {
		if !opts.Force {
			return fmt.Errorf("branch %s is associated with PR #%d. Renaming it will remove this association. Use --force to proceed", currentBranch, *meta.PrInfo.Number)
		}
		splog.Info("Removing association with PR #%d as GitHub PR branch names are immutable.", *meta.PrInfo.Number)
		// Clear PrInfo
		meta.PrInfo = nil
		if err := git.WriteMetadataRef(currentBranch, meta); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	// Take snapshot before modifying the repository
	snapshotOpts := NewSnapshot("rename",
		WithArg(newName),
		WithFlag(opts.Force, "--force"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Perform rename via engine
	oldBranchObj := eng.GetBranch(currentBranch)
	newBranchObj := eng.GetBranch(newName)
	if err := eng.RenameBranch(ctx.Context, oldBranchObj, newBranchObj); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	splog.Info("Renamed %s to %s.", tui.ColorBranchName(currentBranch, false), tui.ColorBranchName(newName, true))

	return nil
}
