package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newScopeCmd creates the scope command
func newScopeCmd() *cobra.Command {
	var (
		unset   bool
		show    bool
		message string
	)

	cmd := &cobra.Command{
		Use:   "scope [name] [message]",
		Short: "Manage the logical scope for the current branch",
		Long: `Manage the logical scope (e.g., Jira Ticket ID, Linear ID) for the current branch.
By default, branches inherit their scope from their parent. Using this command sets an 
explicit override for the current branch and all its descendants.

If on trunk, this command starts a new logical unit of work by creating a new branch 
with the specified scope.

Use 'none' or 'clear' as the scope name to explicitly break the inheritance chain.`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			var scope string
			if len(args) > 0 {
				scope = args[0]
			}

			if len(args) > 1 && message == "" {
				message = args[1]
			}

			if scope == "" && !unset && !show {
				show = true // Default to show if no args/flags
			}

			opts := actions.ScopeOptions{
				Scope:   scope,
				Message: message,
				Unset:   unset,
				Show:    show,
			}

			return actions.ScopeAction(ctx, opts)
		},
	}

	cmd.Flags().BoolVar(&unset, "unset", false, "Remove the explicit scope override from the current branch")
	cmd.Flags().BoolVar(&show, "show", false, "Show the current scope for this branch")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Specify a commit message (starts a new branch if on trunk or if message is provided)")

	return cmd
}
