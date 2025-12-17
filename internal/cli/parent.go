package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// newParentCmd creates the parent command
func newParentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parent",
		Short: "Show the parent of the current branch",
		Long: `Show the parent of the current branch.

Displays the name of the branch that is the parent of the current branch
in the stack. This is useful for understanding the structure of your stack
and seeing which branch the current branch is based on.`,
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
			ctx := runtime.NewContext(eng)

			// Get current branch
			currentBranch := ctx.Engine.CurrentBranch()
			if currentBranch == "" {
				return errors.ErrNotOnBranch
			}

			// Check if on trunk
			if ctx.Engine.IsTrunk(currentBranch) {
				ctx.Splog.Info("%s is trunk and has no parent.", output.ColorBranchName(currentBranch, true))
				return nil
			}

			// Get parent
			parent := ctx.Engine.GetParent(currentBranch)
			if parent == "" {
				ctx.Splog.Info("%s has no parent (untracked branch).", output.ColorBranchName(currentBranch, true))
				return nil
			}

			// Print parent
			fmt.Println(parent)
			return nil
		},
	}

	return cmd
}
