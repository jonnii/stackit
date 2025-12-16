package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newModifyCmd creates the modify command
func newModifyCmd() *cobra.Command {
	var (
		all               bool
		commit            bool
		edit              bool
		interactiveRebase bool
		message           string
		patch             bool
		resetAuthor       bool
		update            bool
		verbose           int
	)

	cmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify the current branch by amending its commit or creating a new commit",
		Long: `Modify the current branch by amending its commit or creating a new commit.

Automatically restacks descendants after the modification.

If you have any unstaged changes, you will be asked whether you'd like to stage them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = all               // Will be used when implemented
			_ = commit            // Will be used when implemented
			_ = edit              // Will be used when implemented
			_ = interactiveRebase // Will be used when implemented
			_ = message           // Will be used when implemented
			_ = patch             // Will be used when implemented
			_ = resetAuthor       // Will be used when implemented
			_ = update            // Will be used when implemented
			_ = verbose           // Will be used when implemented
			return fmt.Errorf("modify command not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stage all changes before committing.")
	cmd.Flags().BoolVarP(&commit, "commit", "c", false, "Create a new commit instead of amending the current commit. If this branch has no commits, this command always creates a new commit.")
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "If passed, open an editor to edit the commit message. When creating a new commit, this flag is ignored.")
	cmd.Flags().BoolVar(&interactiveRebase, "interactive-rebase", false, "Ignore all other flags and start a git interactive rebase on the commits in this branch.")
	cmd.Flags().StringVarP(&message, "message", "m", "", "The message for the new or amended commit. If passed, no editor is opened.")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Pick hunks to stage before committing.")
	cmd.Flags().BoolVar(&resetAuthor, "reset-author", false, "Set the author of the commit to the current user if amending.")
	cmd.Flags().BoolVarP(&update, "update", "u", false, "Stage all updates to tracked files before committing.")
	cmd.Flags().CountVarP(&verbose, "verbose", "v", "Show unified diff between the HEAD commit and what would be committed at the bottom of the commit message template. If specified twice, show in addition the unified diff between what would be committed and the worktree files, i.e. the unstaged changes to tracked files.")

	return cmd
}
