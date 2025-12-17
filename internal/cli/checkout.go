package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
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
			// Ensure stackit is initialized
			repoRoot, err := EnsureInitialized()
			if err != nil {
				return err
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := runtime.NewContext(eng)

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
