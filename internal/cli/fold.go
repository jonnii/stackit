package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	_ "stackit.dev/stackit/internal/demo" // Register demo engine factory
	"stackit.dev/stackit/internal/runtime"
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
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run fold action
			return actions.FoldAction(ctx, actions.FoldOptions{
				Keep: keep,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&keep, "keep", "k", false, "Keeps the name of the current branch instead of using the name of its parent.")

	return cmd
}
