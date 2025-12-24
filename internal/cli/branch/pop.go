// Package branch provides CLI commands for managing branches in a stack.
package branch

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	_ "stackit.dev/stackit/internal/demo" // Register demo engine factory
	"stackit.dev/stackit/internal/runtime"
)

// NewPopCmd creates the pop command
func NewPopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pop",
		Short: "Delete the current branch but retain the state of files in the working tree",
		Long: `Delete the current branch but retain the state of files in the working tree.

This is useful when you want to remove a branch from the stack but keep
your uncommitted changes. The working tree will remain unchanged after
the branch is deleted.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run pop action
			return actions.PopAction(ctx, actions.PopOptions{})
		},
	}

	return cmd
}
