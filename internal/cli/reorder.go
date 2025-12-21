package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newReorderCmd creates the reorder command
func newReorderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reorder",
		Short: "Reorder branches between trunk and the current branch",
		Long: `Reorder branches between trunk and the current branch, restacking all of their descendants.

Opens an editor where you can reorder branches by moving around a line
corresponding to each branch. After saving and closing the editor, the
branches will be restacked in the new order.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run reorder action
			return actions.ReorderAction(ctx)
		},
	}

	return cmd
}
