// Package cli provides command-line interface definitions using Cobra,
// including all subcommands and their flag definitions.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "stackit",
		Short: "Stackit is a command line tool that makes working with stacked changes fast & intuitive",
		Long: `Stackit is a command line tool that makes working with stacked changes fast & intuitive.

https://github.com/jonnii/stackit`,
	}

	// Add subcommands
	rootCmd.AddCommand(newAbortCmd())
	rootCmd.AddCommand(newAbsorbCmd())
	rootCmd.AddCommand(newBottomCmd())
	rootCmd.AddCommand(newCheckoutCmd())
	rootCmd.AddCommand(newChildrenCmd())
	rootCmd.AddCommand(newContinueCmd())
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newDownCmd())
	rootCmd.AddCommand(newFoldCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newLogCmd())
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

	return rootCmd
}
