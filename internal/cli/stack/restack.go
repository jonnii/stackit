// Package stack provides CLI commands for operating on entire stacks.
package stack

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// NewRestackCmd creates the restack command
func NewRestackCmd() *cobra.Command {
	var (
		branch    string
		downstack bool
		only      bool
		upstack   bool
	)

	cmd := &cobra.Command{
		Use:   "restack",
		Short: "Ensure each branch in the current stack has its parent in its Git commit history, rebasing if necessary",
		Long: `Ensure each branch in the current stack has its parent in its Git commit history, rebasing if necessary.
If conflicts are encountered, you will be prompted to resolve them via an interactive Git rebase.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Validation: only one scope flag at a time
			scopeFlags := 0
			if downstack {
				scopeFlags++
			}
			if only {
				scopeFlags++
			}
			if upstack {
				scopeFlags++
			}
			if scopeFlags > 1 {
				return fmt.Errorf("only one of --downstack, --only, or --upstack can be specified")
			}

			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Determine target branch
			targetBranch := branch
			if targetBranch == "" {
				currentBranch := ctx.Engine.CurrentBranch()
				if currentBranch == nil {
					return fmt.Errorf("not on a branch and --branch not specified")
				}
				targetBranch = currentBranch.GetName()
			}

			// Determine scope based on flags
			rng := engine.StackRange{
				RecursiveParents:  !only && !upstack,   // Default or downstack
				IncludeCurrent:    true,                // Always include current
				RecursiveChildren: !only && !downstack, // Default or upstack
			}

			// Run restack action
			return actions.RestackAction(ctx, actions.RestackOptions{
				BranchName: targetBranch,
				Scope:      rng,
			})
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Which branch to run this command from. Defaults to the current branch.")
	cmd.Flags().BoolVar(&downstack, "downstack", false, "Only restack this branch and its ancestors.")
	cmd.Flags().BoolVar(&only, "only", false, "Only restack this branch.")
	cmd.Flags().BoolVar(&upstack, "upstack", false, "Only restack this branch and its descendants.")

	return cmd
}
