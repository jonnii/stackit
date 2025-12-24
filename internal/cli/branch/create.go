// Package branch provides CLI commands for managing branches in a stack.
package branch

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/create"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/runtime"
)

// NewCreateCmd creates the create command
func NewCreateCmd() *cobra.Command {
	var (
		all     bool
		insert  bool
		message string
		patch   bool
		scope   string
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
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get branch name from args
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			}

			// Get config values
			branchPattern, _ := config.GetBranchPattern(ctx.RepoRoot)

			// Prepare options
			opts := create.Options{
				BranchName:    branchName,
				Message:       message,
				Scope:         scope,
				All:           all,
				Insert:        insert,
				Patch:         patch,
				Update:        update,
				Verbose:       verbose,
				BranchPattern: branchPattern,
			}

			// Execute create action
			return create.Action(ctx, opts)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Stage all unstaged changes before creating the branch, including to untracked files")
	cmd.Flags().BoolVarP(&insert, "insert", "i", false, "Insert this branch between the current branch and its child. If there are multiple children, prompts you to select which should be moved onto the new branch")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Specify a commit message")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Pick hunks to stage before committing")
	cmd.Flags().StringVar(&scope, "scope", "", "Set a scope (e.g., Jira ticket ID, Linear ID) for the new branch. If not provided, inherits from parent branch")
	cmd.Flags().BoolVarP(&update, "update", "u", false, "Stage all updates to tracked files before creating the branch")
	cmd.Flags().CountVarP(&verbose, "verbose", "v", "Show unified diff between the HEAD commit and what would be committed at the bottom of the commit message template. If specified twice, show in addition the unified diff between what would be committed and the worktree files")

	return cmd
}
