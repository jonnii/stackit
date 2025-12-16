package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newFoldCmd creates the fold command
func newFoldCmd() *cobra.Command {
	var (
		keep bool
	)

	cmd := &cobra.Command{
		Use:   "fold",
		Short: "Fold a branch's changes into its parent",
		Long: `Fold a branch's changes into its parent, update dependencies of descendants
of the new combined branch, and restack.

This is useful when you have a branch that is no longer needed and you want to
combine its changes with its parent branch.

This command does not perform any action on GitHub or the remote repository.
If you fold a branch with an open pull request, you will need to manually
close the pull request.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = keep // Will be used when implemented
			return fmt.Errorf("fold command not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&keep, "keep", "k", false, "Keeps the name of the current branch instead of using the name of its parent.")

	return cmd
}
