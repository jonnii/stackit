package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newPopCmd creates the pop command
func newPopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pop",
		Short: "Delete the current branch but retain the state of files in the working tree",
		Long: `Delete the current branch but retain the state of files in the working tree.

This is useful when you want to remove a branch from the stack but keep
your uncommitted changes. The working tree will remain unchanged after
the branch is deleted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("pop command not yet implemented")
		},
	}

	return cmd
}
