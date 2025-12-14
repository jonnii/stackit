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

	return rootCmd
}
