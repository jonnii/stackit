package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newUndoCmd creates the undo command
func newUndoCmd() *cobra.Command {
	var snapshotID string

	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Restore the repository to a previous state",
		Long: `Restore the repository to a previous state before a Stackit command was executed.

This command shows an interactive list of available undo points. Each undo point
represents the state of the repository before a modifying Stackit command (like
'move', 'create', 'restack', etc.) was executed.

If you specify a snapshot ID with --snapshot, it will restore to that specific
state without prompting.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run undo action
			return actions.UndoAction(ctx, actions.UndoOptions{
				SnapshotID: snapshotID,
			})
		},
	}

	// Add flags
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Specific snapshot ID to restore (skips interactive selection)")

	return cmd
}
