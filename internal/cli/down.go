package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDownCmd creates the down command
func newDownCmd() *cobra.Command {
	var (
		steps int
	)

	cmd := &cobra.Command{
		Use:   "down [steps]",
		Short: "Switch to the parent of the current branch",
		Long: `Switch to the parent of the current branch.

Navigates down the stack toward trunk by switching to the parent branch.
By default, moves one level down. Use the --steps flag to move multiple
levels at once.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = steps // Will be used when implemented
			return fmt.Errorf("down command not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse downstack.")

	return cmd
}
