package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
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
			// Initialize git repository
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			// Get repo root
			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			// Auto-initialize if not initialized
			if !config.IsInitialized(repoRoot) {
				splog := output.NewSplog()
				splog.Info("Stackit has not been initialized, attempting to setup now...")

				// Run init logic
				branchNames, err := git.GetAllBranchNames()
				if err != nil {
					return fmt.Errorf("failed to get branches: %w", err)
				}

				if len(branchNames) == 0 {
					return fmt.Errorf("no branches found in current repo; cannot initialize Stackit.\nPlease create your first commit and then re-run stackit init")
				}

				// Infer trunk
				trunkName := InferTrunk(branchNames)
				if trunkName == "" {
					// Fallback to first branch or main
					trunkName = "main"
					found := false
					for _, name := range branchNames {
						if name == "main" {
							found = true
							break
						}
					}
					if !found && len(branchNames) > 0 {
						trunkName = branchNames[0]
					}
				}

				if err := config.SetTrunk(repoRoot, trunkName); err != nil {
					return fmt.Errorf("failed to initialize: %w", err)
				}
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := context.NewContext(eng)

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
