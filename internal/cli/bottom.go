package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// newBottomCmd creates the bottom command
func newBottomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bottom",
		Short: "Switch to the branch closest to trunk in the current stack",
		Long: `Switch to the branch closest to trunk in the current stack.

This command navigates down the parent chain from the current branch until
it reaches the first branch that has trunk as its parent (or trunk itself).`,
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

			// Auto-initialize if not initialized
			if !config.IsInitialized(repoRoot) {
				splog := output.NewSplog()
				splog.Info("Stackit has not been initialized, attempting to setup now...")

				// Run init logic
				branchNames, err := git.GetAllBranchNames()
				if err != nil {
					return fmt.Errorf("failed to get branches: %w", err)
				}

				if len(branchNames) == 0 {
					return fmt.Errorf("no branches found in current repo; cannot initialize Stackit.\nPlease create your first commit and then re-run stackit init")
				}

				// Infer trunk
				trunkName := InferTrunk(branchNames)
				if trunkName == "" {
					// Fallback to first branch or main
					trunkName = "main"
					found := false
					for _, name := range branchNames {
						if name == "main" {
							found = true
							break
						}
					}
					if !found && len(branchNames) > 0 {
						trunkName = branchNames[0]
					}
				}

				if err := config.SetTrunk(repoRoot, trunkName); err != nil {
					return fmt.Errorf("failed to initialize: %w", err)
				}
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := context.NewContext(eng)

			// Execute bottom action
			return actions.SwitchBranchAction(actions.DirectionBottom, ctx)
		},
	}

	return cmd
}
