package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			_ = force // Will be used when implemented
			return fmt.Errorf("abort command not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Do not prompt for confirmation; abort immediately.")

	return cmd
}
