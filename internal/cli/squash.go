package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// newSquashCmd creates the squash command
func newSquashCmd() *cobra.Command {
	var (
		message string
		edit    bool
		noEdit  bool
	)

	cmd := &cobra.Command{
		Use:   "squash",
		Short: "Squash all commits in the current branch into a single commit and restack upstack branches",
		Long: `Squash all commits in the current branch into a single commit and restack upstack branches.

This command combines all commits in the current branch into a single commit. After squashing,
all upstack branches (children) are automatically restacked.`,
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
			ctx := context.NewContext(eng)

			// Determine noEdit flag: --no-edit takes precedence over --edit
			// If --no-edit is set, noEdit = true
			// If --message is provided, noEdit = true (message provided, don't edit)
			// Otherwise, noEdit = false (default to editing)
			noEditFlag := noEdit
			if !noEditFlag && message != "" {
				noEditFlag = true // Message provided, don't edit
			}

			// Run squash action
			return actions.SquashAction(actions.SquashOptions{
				Message:  message,
				NoEdit:   noEditFlag,
				Engine:   eng,
				Splog:    ctx.Splog,
				RepoRoot: repoRoot,
			})
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "The updated message for the commit.")
	cmd.Flags().BoolVar(&edit, "edit", true, "Modify the existing commit message.")
	cmd.Flags().BoolVarP(&noEdit, "no-edit", "n", false, "Don't modify the existing commit message. Takes precedence over --edit")

	return cmd
}
