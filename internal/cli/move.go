package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newMoveCmd creates the move command
func newMoveCmd() *cobra.Command {
	var (
		all    bool
		onto   string
		source string
	)

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Rebase the current branch onto the target branch",
		Long: `Rebase the current branch onto the target branch and restack all of its descendants.

If no branch is passed in, opens an interactive selector to choose the target branch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = all    // Will be used when implemented
			_ = onto   // Will be used when implemented
			_ = source // Will be used when implemented
			return fmt.Errorf("move command not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show branches across all configured trunks in interactive selection.")
	cmd.Flags().StringVarP(&onto, "onto", "o", "", "Branch to move the current branch onto.")
	cmd.Flags().StringVar(&source, "source", "", "Branch to move (defaults to current branch).")

	return cmd
}
