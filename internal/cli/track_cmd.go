package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/runtime"
)

// newTrackCmd creates the track command
func newTrackCmd() *cobra.Command {
	var (
		force  bool
		parent string
	)

	cmd := &cobra.Command{
		Use:   "track [branch]",
		Short: "Start tracking a branch with stackit by selecting its parent",
		Long: `Start tracking the current (or provided) branch with stackit by selecting its parent.
Can recursively track a stack of branches by specifying each branch's parent interactively.
This command can also be used to fix corrupted stackit metadata.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get branch name from args or use current branch
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			} else {
				branchName = ctx.Engine.CurrentBranch()
				if branchName == "" {
					return errors.ErrNotOnBranch
				}
			}

			// Execute track action
			return actions.TrackAction(ctx, actions.TrackOptions{
				BranchName: branchName,
				Force:      force,
				Parent:     parent,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Sets the parent to the most recent tracked ancestor of the branch being tracked to skip prompts. Takes precedence over --parent")
	cmd.Flags().StringVarP(&parent, "parent", "p", "", "The tracked branch's parent. Must be set to a tracked branch. If provided, only one branch can be tracked at a time.")

	return cmd
}
