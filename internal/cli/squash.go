package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newSquashCmd creates the squash command
func newSquashCmd() *cobra.Command {
	var (
		message string
		edit    bool
		noEdit  bool
	)

	cmd := &cobra.Command{
		Use:   "squash",
		Short: "Squash all commits in the current branch into a single commit and restack upstack branches",
		Long: `Squash all commits in the current branch into a single commit and restack upstack branches.

This command combines all commits in the current branch into a single commit. After squashing,
all upstack branches (children) are automatically restacked.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Determine noEdit flag: matches charcoal logic: noEdit = argv['no-edit'] || !argv.edit
			// If --no-edit is set, noEdit = true
			// If --edit is false, noEdit = true
			// Otherwise (edit is true, the default), noEdit = false (will edit)
			noEditFlag := noEdit || !edit

			// Run squash action
			return actions.SquashAction(ctx, actions.SquashOptions{
				Message: message,
				NoEdit:  noEditFlag,
			})
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "The updated message for the commit.")
	cmd.Flags().BoolVar(&edit, "edit", true, "Modify the existing commit message.")
	cmd.Flags().BoolVarP(&noEdit, "no-edit", "n", false, "Don't modify the existing commit message. Takes precedence over --edit")

	return cmd
}
