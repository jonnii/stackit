// Package cli provides command-line interface definitions using Cobra,
// including all subcommands and their flag definitions.
package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/cli/branch"
	"stackit.dev/stackit/internal/cli/navigation"
	"stackit.dev/stackit/internal/cli/stack"
)

// NewRootCmd creates the root cobra command
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "stackit",
		Short:   "Stackit is a command line tool that makes working with stacked changes fast & intuitive",
		Version: version,
		Long: `Stackit is a command line tool that makes working with stacked changes fast & intuitive.

https://github.com/jonnii/stackit

Version: ` + version + `
Commit:  ` + commit + `
Date:    ` + date,
	}

	// Add subcommands
	rootCmd.AddCommand(newAbortCmd())
	rootCmd.AddCommand(branch.NewAbsorbCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(navigation.NewBottomCmd())
	rootCmd.AddCommand(navigation.NewCheckoutCmd())
	rootCmd.AddCommand(navigation.NewChildrenCmd())
	rootCmd.AddCommand(newContinueCmd())
	rootCmd.AddCommand(branch.NewCreateCmd())
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(branch.NewDeleteCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(navigation.NewDownCmd())
	rootCmd.AddCommand(branch.NewFoldCmd())
	rootCmd.AddCommand(stack.NewForeachCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(navigation.NewLogCmd())
	rootCmd.AddCommand(stack.NewMergeCmd())
	rootCmd.AddCommand(branch.NewModifyCmd())
	rootCmd.AddCommand(stack.NewMoveCmd())
	rootCmd.AddCommand(navigation.NewParentCmd())
	rootCmd.AddCommand(branch.NewPopCmd())
	rootCmd.AddCommand(branch.NewRenameCmd())
	rootCmd.AddCommand(stack.NewReorderCmd())
	rootCmd.AddCommand(stack.NewRestackCmd())
	rootCmd.AddCommand(branch.NewSplitCmd())
	rootCmd.AddCommand(branch.NewSquashCmd())
	rootCmd.AddCommand(newScopeCmd())
	rootCmd.AddCommand(stack.NewSubmitCmd())
	rootCmd.AddCommand(stack.NewSyncCmd())
	rootCmd.AddCommand(navigation.NewTopCmd())
	rootCmd.AddCommand(newTrackCmd())
	rootCmd.AddCommand(newUntrackCmd())
	rootCmd.AddCommand(navigation.NewTrunkCmd())
	rootCmd.AddCommand(newUndoCmd())
	rootCmd.AddCommand(navigation.NewUpCmd())
	rootCmd.AddCommand(newConfigCmd())

	// Add aliases
	rootCmd.AddCommand(stack.NewSsCmd())

	return rootCmd
}
