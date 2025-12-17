package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/demo"
	"stackit.dev/stackit/internal/engine"
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
			// Check for demo mode
			if ctx, ok := demo.NewDemoContext(); ok {
				branchName := ctx.Engine.Trunk()
				if stack || steps > 0 {
					branchName = ctx.Engine.CurrentBranch()
				}

				opts := actions.LogOptions{
					Style:         "FULL",
					Reverse:       reverse,
					BranchName:    branchName,
					ShowUntracked: showUntracked,
				}

				if steps > 0 {
					opts.Steps = &steps
				}

				return actions.LogAction(opts, ctx)
			}

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
			return actions.LogAction(opts, ctx)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&reverse, "reverse", "r", false, "Print the log upside down. Handy when you have a lot of branches!")
	cmd.Flags().BoolVarP(&stack, "stack", "s", false, "Only show ancestors and descendants of the current branch")
	cmd.Flags().IntVarP(&steps, "steps", "n", 0, "Only show this many levels upstack and downstack. Implies --stack")
	cmd.Flags().BoolVarP(&showUntracked, "show-untracked", "u", false, "Include untracked branches in interactive selection")

	return cmd
}
