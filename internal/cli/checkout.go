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

// newCheckoutCmd creates the checkout command
func newCheckoutCmd() *cobra.Command {
	var (
		all           bool
		showUntracked bool
		stack         bool
		trunk         bool
	)

	cmd := &cobra.Command{
		Use:     "checkout [branch]",
		Aliases: []string{"co"},
		Short:   "Switch to a branch. If no branch is provided, opens an interactive selector.",
		Long: `Switch to a branch. If no branch is provided, opens an interactive selector.

The interactive selector allows you to navigate branches using arrow keys and filter
by typing. Use flags to customize which branches are shown.`,
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

			// Get branch name from args
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			}

			// Prepare options
			opts := actions.CheckoutOptions{
				BranchName:    branchName,
				ShowUntracked: showUntracked,
				All:           all,
				StackOnly:     stack,
				CheckoutTrunk: trunk,
			}

			// Execute checkout action
			return actions.CheckoutAction(opts, ctx)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show branches across all configured trunks in interactive selection")
	cmd.Flags().BoolVarP(&showUntracked, "show-untracked", "u", false, "Include untracked branches in interactive selection")
	cmd.Flags().BoolVarP(&stack, "stack", "s", false, "Only show ancestors and descendants of the current branch in interactive selection")
	cmd.Flags().BoolVarP(&trunk, "trunk", "t", false, "Checkout the current trunk")

	return cmd
}
