package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// newChildrenCmd creates the children command
func newChildrenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "children",
		Short: "Show the children of the current branch",
		Long: `Show the children of the current branch.

Lists all branches that have the current branch as their parent in the stack.
This is useful for understanding the structure of your stack and seeing which
branches depend on the current branch.`,
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
				return fmt.Errorf("stackit is not initialized. Run 'stackit init' first")
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := context.NewContext(eng)

			// Get current branch
			currentBranch := ctx.Engine.CurrentBranch()
			if currentBranch == "" {
				return errors.ErrNotOnBranch
			}

			// Get children
			children := ctx.Engine.GetChildren(currentBranch)
			if len(children) == 0 {
				ctx.Splog.Info("%s has no children.", output.ColorBranchName(currentBranch, true))
				return nil
			}

			// Print children
			for _, child := range children {
				fmt.Println(child)
			}
			return nil
		},
	}

	return cmd
}
