package main

import (
	"os"

	"stackit.dev/stackit/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Check for passthrough commands before processing with cobra
	if cli.HandlePassthrough(os.Args) {
		return // HandlePassthrough already exited
	}

	rootCmd := cli.NewRootCmd(version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
