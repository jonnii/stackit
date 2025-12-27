// Package navigation provides CLI commands for navigating branches in a stack.
package navigation

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/runtime"
)

// NewLogCmd creates the log command
func NewLogCmd() *cobra.Command {
	f := &logFlags{}

	cmd := &cobra.Command{
		Use:          "log",
		Short:        "Log all branches tracked by Stackit, showing dependencies and info for each",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeLog(cmd, f, "NORMAL")
		},
	}

	addLogFlags(cmd, f)

	// Add subcommands
	cmd.AddCommand(newLogFullCmd())

	return cmd
}

func newLogFullCmd() *cobra.Command {
	f := &logFlags{}
	cmd := &cobra.Command{
		Use:          "full",
		Short:        "Log branches with GitHub state (PR status, CI checks)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeLog(cmd, f, "FULL")
		},
	}
	addLogFlags(cmd, f)
	return cmd
}

type logFlags struct {
	reverse       bool
	stack         bool
	steps         int
	showUntracked bool
}

func addLogFlags(cmd *cobra.Command, f *logFlags) {
	cmd.Flags().BoolVarP(&f.reverse, "reverse", "r", false, "Print the log upside down. Handy when you have a lot of branches!")
	cmd.Flags().BoolVarP(&f.stack, "stack", "s", false, "Only show ancestors and descendants of the current branch")
	cmd.Flags().IntVarP(&f.steps, "steps", "n", 0, "Only show this many levels upstack and downstack. Implies --stack")
	cmd.Flags().BoolVarP(&f.showUntracked, "show-untracked", "u", false, "Include untracked branches in interactive selection")
}

func executeLog(cmd *cobra.Command, f *logFlags, style string) error {
	return common.Run(cmd, func(ctx *runtime.Context) error {
		eng := ctx.Engine

		// Determine branch name
		trunk := eng.Trunk()
		branchName := trunk.GetName()
		if f.stack || f.steps > 0 {
			currentBranch := eng.CurrentBranch()
			if currentBranch == nil {
				return fmt.Errorf("not on a branch")
			}
			branchName = currentBranch.GetName()
		}

		// Prepare options
		opts := actions.LogOptions{
			Style:         style,
			Reverse:       f.reverse,
			BranchName:    branchName,
			ShowUntracked: f.showUntracked,
		}

		if f.steps > 0 {
			opts.Steps = &f.steps
		}

		// Execute log action
		return actions.LogAction(ctx, opts)
	})
}
