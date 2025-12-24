package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
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
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run continue action
			return actions.ContinueAction(ctx, actions.ContinueOptions{
				AddAll: addAll,
			})
		},
	}

	cmd.Flags().BoolVarP(&addAll, "all", "a", false, "Stage all changes before continuing")

	return cmd
}
