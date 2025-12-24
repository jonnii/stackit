package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newDebugCmd creates the debug command
func newDebugCmd() *cobra.Command {
	var (
		limit int
	)

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Dump debugging information about recent commands and stack state",
		Long: `Dump comprehensive debugging information including recent command history
and complete stack state. This is useful for diagnosing issues when stacks
get into a bad state.

The output includes:
  - Recent commands and their parameters (from undo snapshots)
  - Complete stack state (branches, relationships, metadata, PR info)
  - Continuation state (if exists)
  - Repository information

Output is formatted as pretty-printed JSON for easy reading and parsing.`,
		Args: cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run debug action
			return actions.DebugAction(ctx, actions.DebugOptions{
				Limit: limit,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of recent commands to show (0 = all)")

	return cmd
}
