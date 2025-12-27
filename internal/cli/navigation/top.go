package navigation

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/cli/helpers"
	"stackit.dev/stackit/internal/runtime"
)

// NewTopCmd creates the top command
func NewTopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Switch to the tip branch of the current stack",
		Long: `Switch to the tip branch of the current stack. Prompts if ambiguous.

This command navigates up the children chain from the current branch until
it reaches a branch with no children (the tip of the stack). If multiple
children exist at any level, you will be prompted to select which branch
to follow.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return helpers.Run(cmd, func(ctx *runtime.Context) error {
				// Execute top action
				return actions.SwitchBranchAction(actions.DirectionTop, ctx)
			})
		},
	}

	return cmd
}
