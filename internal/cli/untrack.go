package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/runtime"
)

// newUntrackCmd creates the untrack command
func newUntrackCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "untrack [branch]",
		Short: "Stop tracking a branch with stackit",
		Long: `Stop tracking the current (or provided) branch with stackit.
If the branch has children, they will also be untracked.`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeBranches,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get branch name from args or use current branch
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			} else {
				currentBranch := ctx.Engine.CurrentBranch()
				if currentBranch.Name == "" {
					return errors.ErrNotOnBranch
				}
				branchName = currentBranch.Name
			}

			// Execute untrack action
			return actions.UntrackAction(ctx, actions.UntrackOptions{
				BranchName: branchName,
				Force:      force,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Will not prompt for confirmation before untracking a branch with children")

	return cmd
}
