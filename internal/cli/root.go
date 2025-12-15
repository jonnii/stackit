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
	rootCmd.AddCommand(newLogCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newBottomCmd())
	rootCmd.AddCommand(newTopCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newSubmitCmd())
	rootCmd.AddCommand(newMergeCmd())
	rootCmd.AddCommand(newRestackCmd())
	rootCmd.AddCommand(newContinueCmd())
	rootCmd.AddCommand(newSquashCmd())
	rootCmd.AddCommand(newSplitCmd())

	return rootCmd
}
