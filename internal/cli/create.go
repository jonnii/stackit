package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// newCreateCmd creates the create command
func newCreateCmd() *cobra.Command {
	var (
		all     bool
		insert  bool
		message string
		patch   bool
		update  bool
		verbose int
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new branch stacked on top of the current branch",
		Long: `Create a new branch stacked on top of the current branch and commit staged changes.

If no branch name is specified, generate a branch name from the commit message.
If your working directory contains no changes, an empty branch will be created.
If you have any unstaged changes, you will be asked whether you'd like to stage them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure stackit is initialized
			repoRoot, err := EnsureInitialized()
			if err != nil {
				return err
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := runtime.NewContext(eng)

			// Get branch name from args
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			}

			// Prepare options
			opts := actions.CreateOptions{
				BranchName: branchName,
				Message:    message,
				All:        all,
				Insert:     insert,
				Patch:      patch,
				Update:     update,
				Verbose:    verbose,
			}

			// Execute create action
			return actions.CreateAction(opts, ctx)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stage all unstaged changes before creating the branch, including to untracked files")
	cmd.Flags().BoolVarP(&insert, "insert", "i", false, "Insert this branch between the current branch and its child. If there are multiple children, prompts you to select which should be moved onto the new branch")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Specify a commit message")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Pick hunks to stage before committing")
	cmd.Flags().BoolVarP(&update, "update", "u", false, "Stage all updates to tracked files before creating the branch")
	cmd.Flags().CountVarP(&verbose, "verbose", "v", "Show unified diff between the HEAD commit and what would be committed at the bottom of the commit message template. If specified twice, show in addition the unified diff between what would be committed and the worktree files")

	return cmd
}
