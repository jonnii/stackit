// Package branch provides CLI commands for managing branches in a stack.
package branch

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/absorb"
	"stackit.dev/stackit/internal/runtime"
)

// NewAbsorbCmd creates the absorb command
func NewAbsorbCmd() *cobra.Command {
	var (
		all    bool
		dryRun bool
		force  bool
		patch  bool
	)

	cmd := &cobra.Command{
		Use:   "absorb",
		Short: "Amend staged changes to the relevant commits in the current stack",
		Long: `Amend staged changes to the relevant commits in the current stack.

Relevance is calculated by checking the changes in each commit downstack from the current commit,
and finding the first commit that each staged hunk (consecutive lines of changes) can be applied to deterministically.
If there is no clear commit to absorb a hunk into, it will not be absorbed.

Prompts for confirmation before amending the commits, and restacks the branches upstack of the current branch.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Run absorb action
			return absorb.Action(ctx, absorb.Options{
				All:    all,
				DryRun: dryRun,
				Force:  force,
				Patch:  patch,
			})
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stage all unstaged changes before absorbing. Unlike create and modify, this will not include untracked files, as file creations would never be absorbed.")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Print which commits the hunks would be absorbed into, but do not actually absorb them.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Do not prompt for confirmation; apply the hunks to the commits immediately.")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Pick hunks to stage before absorbing.")

	return cmd
}
