package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newChildrenCmd creates the children command
func newChildrenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "children",
		Short: "Show the children of the current branch",
		Long: `Show the children of the current branch.

Lists all branches that have the current branch as their parent in the stack.
This is useful for understanding the structure of your stack and seeing which
branches depend on the current branch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("children command not yet implemented")
		},
	}

	return cmd
}
