// Package undo provides functionality for undoing stackit operations using snapshots.
package undo

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/timeutil"
	"stackit.dev/stackit/internal/tui"
)

// Options contains options for the undo command
type Options struct {
	SnapshotID string // Optional: specific snapshot to restore (skips interactive selection)
}

// Action performs the undo operation
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get all available snapshots
	snapshots, err := eng.GetSnapshots()
	if err != nil {
		return fmt.Errorf("failed to get snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		splog.Info("No undo history available.")
		return nil
	}

	var selectedSnapshotID string

	// If snapshot ID is provided, use it directly
	if opts.SnapshotID != "" {
		// Verify the snapshot exists
		found := false
		for _, snap := range snapshots {
			if snap.ID == opts.SnapshotID {
				found = true
				selectedSnapshotID = snap.ID
				break
			}
		}
		if !found {
			return fmt.Errorf("snapshot %s not found", opts.SnapshotID)
		}
	} else {
		// Interactive selection
		if len(snapshots) == 1 {
			// Only one snapshot, use it directly
			selectedSnapshotID = snapshots[0].ID
			splog.Info("Restoring to: %s", snapshots[0].DisplayName)
		} else {
			// Multiple snapshots - show interactive selector
			options := make([]tui.SelectOption, len(snapshots))
			for i, snap := range snapshots {
				options[i] = tui.SelectOption{
					Label: snap.DisplayName,
					Value: snap.ID,
				}
			}

			selected, err := tui.PromptSelect("Select state to restore:", options, 0)
			if err != nil {
				return fmt.Errorf("failed to select snapshot: %w", err)
			}

			selectedSnapshotID = selected
		}
	}

	// Find the selected snapshot info for display
	var selectedSnapshot *engine.SnapshotInfo
	for _, snap := range snapshots {
		if snap.ID == selectedSnapshotID {
			selectedSnapshot = &snap
			break
		}
	}

	if selectedSnapshot == nil {
		return fmt.Errorf("selected snapshot not found")
	}

	// Count how many snapshots will be "undone" (all snapshots newer than the selected one)
	undoneCount := 0
	for _, snap := range snapshots {
		if snap.Timestamp.After(selectedSnapshot.Timestamp) {
			undoneCount++
		}
	}

	// Show confirmation prompt
	confirmMessage := fmt.Sprintf(
		"This will restore the repository to the state before '%s' (%s).",
		selectedSnapshot.Command,
		timeutil.FormatTimeAgo(selectedSnapshot.Timestamp),
	)
	if undoneCount > 0 {
		confirmMessage += fmt.Sprintf(" This will undo %d subsequent action(s).", undoneCount)
	}
	confirmMessage += " Are you sure?"

	confirmed, err := tui.PromptConfirm(confirmMessage, false)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmed {
		splog.Info("Undo canceled.")
		return nil
	}

	// Abort any in-progress Git operations that might interfere with restoration
	if git.IsRebaseInProgress(ctx.Context) {
		splog.Info("Aborting in-progress rebase before undo...")
		if err := git.RebaseAbort(ctx.Context); err != nil {
			return fmt.Errorf("failed to abort rebase: %w", err)
		}
	}
	if git.IsMergeInProgress(ctx.Context) {
		splog.Info("Aborting in-progress merge before undo...")
		if err := git.MergeAbort(ctx.Context); err != nil {
			return fmt.Errorf("failed to abort merge: %w", err)
		}
	}

	// Perform the restoration
	splog.Info("Restoring repository state...")
	if err := eng.RestoreSnapshot(ctx.Context, selectedSnapshotID); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	splog.Info("Successfully restored to state before '%s'.", selectedSnapshot.Command)

	return nil
}
