// Package cli provides command-line interface definitions using Cobra,
// including all subcommands and their flag definitions.
package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/cli/navigation"
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
	rootCmd.AddCommand(newAbsorbCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(navigation.NewBottomCmd())
	rootCmd.AddCommand(navigation.NewCheckoutCmd())
	rootCmd.AddCommand(navigation.NewChildrenCmd())
	rootCmd.AddCommand(newContinueCmd())
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(navigation.NewDownCmd())
	rootCmd.AddCommand(newFoldCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(navigation.NewLogCmd())
	rootCmd.AddCommand(newMergeCmd())
	rootCmd.AddCommand(newModifyCmd())
	rootCmd.AddCommand(newMoveCmd())
	rootCmd.AddCommand(navigation.NewParentCmd())
	rootCmd.AddCommand(newPopCmd())
	rootCmd.AddCommand(newReorderCmd())
	rootCmd.AddCommand(newRestackCmd())
	rootCmd.AddCommand(newSplitCmd())
	rootCmd.AddCommand(newSquashCmd())
	rootCmd.AddCommand(newScopeCmd())
	rootCmd.AddCommand(newSubmitCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(navigation.NewTopCmd())
	rootCmd.AddCommand(newTrackCmd())
	rootCmd.AddCommand(newUntrackCmd())
	rootCmd.AddCommand(navigation.NewTrunkCmd())
	rootCmd.AddCommand(newUndoCmd())
	rootCmd.AddCommand(navigation.NewUpCmd())
	rootCmd.AddCommand(newConfigCmd())

	// Add aliases
	rootCmd.AddCommand(navigation.NewLsCmd())
	rootCmd.AddCommand(navigation.NewLlCmd())
	rootCmd.AddCommand(newSsCmd())

	return rootCmd
}
