package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
			return fmt.Errorf("reorder command not yet implemented")
		},
	}

	return cmd
}
