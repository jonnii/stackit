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

https://docs.graphite.dev/guides/graphite-cli`,
	}

	// Add subcommands
	rootCmd.AddCommand(newLogCmd())

	return rootCmd
}
