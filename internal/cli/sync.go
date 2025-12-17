package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
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
			// Initialize git repository
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			// Get repo root
			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			// Check if initialized
			if !config.IsInitialized(repoRoot) {
				return fmt.Errorf("stackit not initialized. Run 'stackit init' first")
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := runtime.NewContext(eng)

			// Handle --all flag (stub for now)
			if all {
				// For now, just sync the current trunk
				// In the future, this would sync across all configured trunks
				ctx.Splog.Info("Syncing branches across all configured trunks...")
			}

			// Run sync action
			return actions.SyncAction(actions.SyncOptions{
				All:     all,
				Force:   force,
				Restack: restack,
				Engine:  eng,
				Splog:   ctx.Splog,
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
