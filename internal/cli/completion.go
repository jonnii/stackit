package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

// completeBranches is a helper for cobra.ValidArgsFunction and RegisterFlagCompletionFunc
// that returns all branch names in the repository.
func completeBranches(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if err := git.InitDefaultRepo(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
