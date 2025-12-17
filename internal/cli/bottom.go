package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
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

			// Execute bottom action
			return actions.SwitchBranchAction(actions.DirectionBottom, ctx)
		},
	}

	return cmd
}
