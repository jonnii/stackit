package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newLogCmd creates the log command
func newLogCmd() *cobra.Command {
	var (
		reverse       bool
		stack         bool
		steps         int
		showUntracked bool
	)

	cmd := &cobra.Command{
		Use:     "log",
		Short:   "Log all branches tracked by Stackit, showing dependencies and info for each",
		Aliases: []string{"l"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext()
			if err != nil {
				return err
			}

			eng := ctx.Engine

			// Determine branch name
			branchName := eng.Trunk()
			if stack || steps > 0 {
				currentBranch := eng.CurrentBranch()
				if currentBranch == "" {
					return fmt.Errorf("not on a branch")
				}
				branchName = currentBranch
			}

			// Prepare options
			opts := actions.LogOptions{
				Style:         "FULL",
				Reverse:       reverse,
				BranchName:    branchName,
				ShowUntracked: showUntracked,
			}

			if steps > 0 {
				opts.Steps = &steps
			}

			// Execute log action
			return actions.LogAction(ctx, opts)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&reverse, "reverse", "r", false, "Print the log upside down. Handy when you have a lot of branches!")
	cmd.Flags().BoolVarP(&stack, "stack", "s", false, "Only show ancestors and descendants of the current branch")
	cmd.Flags().IntVarP(&steps, "steps", "n", 0, "Only show this many levels upstack and downstack. Implies --stack")
	cmd.Flags().BoolVarP(&showUntracked, "show-untracked", "u", false, "Include untracked branches in interactive selection")

	return cmd
}
