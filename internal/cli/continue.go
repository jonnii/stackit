package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// newContinueCmd creates the continue command
func newContinueCmd() *cobra.Command {
	var addAll bool

	cmd := &cobra.Command{
		Use:   "continue",
		Short: "Continues the most recent Stackit command halted by a rebase conflict",
		Long: `Continues the most recent Stackit command halted by a rebase conflict.
This command will continue the rebase and resume restacking remaining branches.`,
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

			// Run continue action
			return actions.ContinueAction(actions.ContinueOptions{
				AddAll:   addAll,
				Engine:   eng,
				Splog:    ctx.Splog,
				RepoRoot: repoRoot,
			})
		},
	}

	cmd.Flags().BoolVarP(&addAll, "all", "a", false, "Stage all changes before continuing")

	return cmd
}
