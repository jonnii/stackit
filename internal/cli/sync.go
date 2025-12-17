package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newSyncCmd creates the sync command
func newSyncCmd() *cobra.Command {
	var (
		all     bool
		force   bool
		restack bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync all branches with remote",
		Long: `Sync all branches with remote, prompting to delete any branches for PRs that have been merged or closed. 
Restacks all branches in your repository that can be restacked without conflicts. 
If trunk cannot be fast-forwarded to match remote, overwrites trunk with the remote version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext()
			if err != nil {
				return err
			}

			// Run sync action
			return actions.SyncAction(ctx, actions.SyncOptions{
				All:     all,
				Force:   force,
				Restack: restack,
			})
		},
	}

	var noRestack bool

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Sync branches across all configured trunks")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Don't prompt for confirmation before overwriting or deleting a branch")
	cmd.Flags().BoolVar(&restack, "restack", true, "Restack any branches that can be restacked without conflicts")
	cmd.Flags().BoolVar(&noRestack, "no-restack", false, "Skip restacking branches")

	// Apply --no-restack flag
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if noRestack {
			restack = false
		}
	}

	return cmd
}
