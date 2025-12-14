package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// newMergeCmd creates the merge command
func newMergeCmd() *cobra.Command {
	var (
		dryRun  bool
		confirm bool
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge the pull requests associated with all branches from trunk to the current branch via Graphite",
		Long: `Merge the pull requests associated with all branches from trunk to the current branch via Graphite.
This command merges PRs for all branches in the stack from trunk up to (and including) the current branch.`,
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
			ctx := context.NewContext(eng)

			// Run merge action
			return actions.MergeAction(actions.MergeOptions{
				DryRun:  dryRun,
				Confirm: confirm,
				Engine:  eng,
				Splog:   ctx.Splog,
			})
		},
	}

	cmd.Flags().BoolVarP(&confirm, "confirm", "c", false, "Asks for confirmation before merging branches. Prompts for confirmation if the local branches differ from remote, regardless of the value of this flag.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Reports the PRs that would be merged and terminates. No branches are merged.")

	return cmd
}
