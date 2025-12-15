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

// newAbsorbCmd creates the absorb command
func newAbsorbCmd() *cobra.Command {
	var (
		all    bool
		dryRun bool
		force  bool
		patch  bool
	)

	cmd := &cobra.Command{
		Use:   "absorb",
		Short: "Amend staged changes to the relevant commits in the current stack",
		Long: `Amend staged changes to the relevant commits in the current stack.

Relevance is calculated by checking the changes in each commit downstack from the current commit,
and finding the first commit that each staged hunk (consecutive lines of changes) can be applied to deterministically.
If there is no clear commit to absorb a hunk into, it will not be absorbed.

Prompts for confirmation before amending the commits, and restacks the branches upstack of the current branch.`,
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

			// Check if initialized FIRST (before any other checks)
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

			// Run absorb action
			return actions.AbsorbAction(actions.AbsorbOptions{
				All:      all,
				DryRun:   dryRun,
				Force:    force,
				Patch:    patch,
				Engine:   eng,
				Splog:    ctx.Splog,
				RepoRoot: repoRoot,
			})
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stage all unstaged changes before absorbing. Unlike create and modify, this will not include untracked files, as file creations would never be absorbed.")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Print which commits the hunks would be absorbed into, but do not actually absorb them.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Do not prompt for confirmation; apply the hunks to the commits immediately.")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Pick hunks to stage before absorbing.")

	return cmd
}
