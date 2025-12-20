// Package cli provides command-line interface definitions using Cobra,
// including all subcommands and their flag definitions.
package cli

import (
	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(newBottomCmd())
	rootCmd.AddCommand(newCheckoutCmd())
	rootCmd.AddCommand(newChildrenCmd())
	rootCmd.AddCommand(newContinueCmd())
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newDebugPromptCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newDownCmd())
	rootCmd.AddCommand(newFoldCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newLogCmd())
	rootCmd.AddCommand(newLsCmd())
	rootCmd.AddCommand(newLlCmd())
	rootCmd.AddCommand(newMergeCmd())
	rootCmd.AddCommand(newModifyCmd())
	rootCmd.AddCommand(newMoveCmd())
	rootCmd.AddCommand(newParentCmd())
	rootCmd.AddCommand(newPopCmd())
	rootCmd.AddCommand(newReorderCmd())
	rootCmd.AddCommand(newRestackCmd())
	rootCmd.AddCommand(newSplitCmd())
	rootCmd.AddCommand(newSquashCmd())
	rootCmd.AddCommand(newSubmitCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newTopCmd())
	rootCmd.AddCommand(newTrunkCmd())
	rootCmd.AddCommand(newConfigCmd())

	return rootCmd
}
