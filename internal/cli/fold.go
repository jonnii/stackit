package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/fold"
	_ "stackit.dev/stackit/internal/demo" // Register demo engine factory
	"stackit.dev/stackit/internal/runtime"
)

// newFoldCmd creates the fold command
func newFoldCmd() *cobra.Command {
	var (
		keep       bool
		allowTrunk bool
	)

	cmd := &cobra.Command{
		Use:   "fold",
		Short: "Fold a branch's changes into its parent",
		Long: `Fold a branch's changes into its parent, update dependencies of descendants
of the new combined branch, and restack.

This is useful when you have a branch that is no longer needed and you want to
combine its changes with its parent branch.

If the parent of the current branch is the trunk (e.g., main), you must provide
the --allow-trunk flag, as this will modify your local trunk branch directly.

This command does not perform any action on GitHub or the remote repository.
If you fold a branch with an open pull request, you will need to manually
close the pull request.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run fold action
			return fold.Action(ctx, fold.Options{
				Keep:       keep,
				AllowTrunk: allowTrunk,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&keep, "keep", "k", false, "Keeps the name of the current branch instead of using the name of its parent.")
	cmd.Flags().BoolVar(&allowTrunk, "allow-trunk", false, "Allows folding into the trunk branch (e.g., main).")

	return cmd
}
