package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// newInfoCmd creates the info command
func newInfoCmd() *cobra.Command {
	var (
		body  bool
		diff  bool
		patch bool
		stat  bool
	)

	cmd := &cobra.Command{
		Use:     "info [branch]",
		Short:   "Display information about the current branch",
		Aliases: []string{"i"},
		Long: `Display information about a branch, including branch relationships,
PR status, and optionally diffs or patches.

If no branch is specified, displays information about the current branch.`,
		Args: cobra.MaximumNArgs(1),
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

			// Check if initialized
			if !config.IsInitialized(repoRoot) {
				return fmt.Errorf("stackit not initialized. Run 'stackit init' first")
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := runtime.NewContext(eng)

			// Determine branch name
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			}

			// Run info action
			return actions.InfoAction(actions.InfoOptions{
				BranchName: branchName,
				Body:       body,
				Diff:       diff,
				Patch:      patch,
				Stat:       stat,
				Engine:     eng,
				Splog:      ctx.Splog,
			})
		},
	}

	cmd.Flags().BoolVarP(&body, "body", "b", false, "Show the PR body, if it exists")
	cmd.Flags().BoolVarP(&diff, "diff", "d", false, "Show the diff between this branch and its parent. Takes precedence over patch")
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Show the changes made by each commit")
	cmd.Flags().BoolVarP(&stat, "stat", "s", false, "Show a diffstat instead of a full diff. Modifies either --patch or --diff. If neither is passed, implies --diff")

	return cmd
}
