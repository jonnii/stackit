// Package helpers provides shared helper functions for CLI commands.
package helpers

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

// CompleteBranches is a helper for cobra.ValidArgsFunction and RegisterFlagCompletionFunc
// that returns all branch names in the repository.
func CompleteBranches(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if err := git.InitDefaultRepo(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
