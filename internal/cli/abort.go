package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/cli/helpers"
	"stackit.dev/stackit/internal/runtime"
)

// newAbortCmd creates the abort command
func newAbortCmd() *cobra.Command {
	var (
		force bool
	)

	cmd := &cobra.Command{
		Use:   "abort",
		Short: "Abort the current stackit command halted by a rebase conflict",
		Long: `Aborts the current stackit command halted by a rebase conflict.

This command cancels any in-progress operation (such as restack, sync, or merge)
that has been paused due to a rebase conflict. Any changes made during the
operation will be rolled back.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return helpers.Run(cmd, func(ctx *runtime.Context) error {
				return actions.AbortAction(ctx, actions.AbortOptions{
					Force: force,
				})
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Do not prompt for confirmation; abort immediately.")

	return cmd
}
