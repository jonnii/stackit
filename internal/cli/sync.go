package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/sync"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run sync action
			return sync.Action(ctx, sync.Options{
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
	cmd.PreRun = func(_ *cobra.Command, _ []string) {
		if noRestack {
			restack = false
		}
	}

	return cmd
}
