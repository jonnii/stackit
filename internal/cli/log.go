package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newLogCmd creates the log command
func newLogCmd() *cobra.Command {
	f := &logFlags{}

	cmd := &cobra.Command{
		Use:     "log",
		Short:   "Log all branches tracked by Stackit, showing dependencies and info for each",
		Aliases: []string{"l"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeLog(cmd, f, "FULL")
		},
	}

	addLogFlags(cmd, f)

	// Add subcommands
	cmd.AddCommand(newLogShortCmd())
	cmd.AddCommand(newLogLongCmd())

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
	// Get context (demo or real)
	ctx, err := runtime.GetContext(cmd.Context())
	if err != nil {
		return err
	}

	eng := ctx.Engine

	// Determine branch name
	branchName := eng.Trunk()
	if f.stack || f.steps > 0 {
		currentBranch := eng.CurrentBranch()
		if currentBranch == "" {
			return fmt.Errorf("not on a branch")
		}
		branchName = currentBranch
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
}

func newLogShortCmd() *cobra.Command {
	f := &logFlags{}
	cmd := &cobra.Command{
		Use:     "short",
		Short:   "Log branches in a compact format",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeLog(cmd, f, "SHORT")
		},
	}
	addLogFlags(cmd, f)
	return cmd
}

func newLogLongCmd() *cobra.Command {
	f := &logFlags{}
	cmd := &cobra.Command{
		Use:     "long",
		Short:   "Log branches with full details",
		Aliases: []string{"ll"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeLog(cmd, f, "FULL")
		},
	}
	addLogFlags(cmd, f)
	return cmd
}

func newLsCmd() *cobra.Command {
	f := &logFlags{}
	cmd := &cobra.Command{
		Use:    "ls",
		Hidden: true, // Hide from main help to avoid clutter, but keep as alias
		Short:  "Alias for log short",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeLog(cmd, f, "SHORT")
		},
	}
	addLogFlags(cmd, f)
	return cmd
}

func newLlCmd() *cobra.Command {
	f := &logFlags{}
	cmd := &cobra.Command{
		Use:    "ll",
		Hidden: true, // Hide from main help to avoid clutter, but keep as alias
		Short:  "Alias for log long",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeLog(cmd, f, "FULL")
		},
	}
	addLogFlags(cmd, f)
	return cmd
}
