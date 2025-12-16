package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newParentCmd creates the parent command
func newParentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parent",
		Short: "Show the parent of the current branch",
		Long: `Show the parent of the current branch.

Displays the name of the branch that is the parent of the current branch
in the stack. This is useful for understanding the structure of your stack
and seeing which branch the current branch is based on.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("parent command not yet implemented")
		},
	}

	return cmd
}
