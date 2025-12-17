package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// newRestackCmd creates the restack command
func newRestackCmd() *cobra.Command {
	var (
		branch    string
		downstack bool
		only      bool
		upstack   bool
	)

	cmd := &cobra.Command{
		Use:   "restack",
		Short: "Ensure each branch in the current stack has its parent in its Git commit history, rebasing if necessary",
		Long: `Ensure each branch in the current stack has its parent in its Git commit history, rebasing if necessary.
If conflicts are encountered, you will be prompted to resolve them via an interactive Git rebase.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validation: only one scope flag at a time
			scopeFlags := 0
			if downstack {
				scopeFlags++
			}
			if only {
				scopeFlags++
			}
			if upstack {
				scopeFlags++
			}
			if scopeFlags > 1 {
				return fmt.Errorf("only one of --downstack, --only, or --upstack can be specified")
			}

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

			// Determine target branch
			targetBranch := branch
			if targetBranch == "" {
				targetBranch = eng.CurrentBranch()
				if targetBranch == "" {
					return fmt.Errorf("not on a branch and --branch not specified")
				}
			}

			// Determine scope based on flags
			scope := engine.Scope{
				RecursiveParents:  !only && !upstack,   // Default or downstack
				IncludeCurrent:    true,                // Always include current
				RecursiveChildren: !only && !downstack, // Default or upstack
			}

			// Run restack action
			return actions.RestackAction(actions.RestackOptions{
				BranchName: targetBranch,
				Scope:      scope,
				Engine:     eng,
				Splog:      ctx.Splog,
				RepoRoot:   repoRoot,
			})
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Which branch to run this command from. Defaults to the current branch.")
	cmd.Flags().BoolVar(&downstack, "downstack", false, "Only restack this branch and its ancestors.")
	cmd.Flags().BoolVar(&only, "only", false, "Only restack this branch.")
	cmd.Flags().BoolVar(&upstack, "upstack", false, "Only restack this branch and its descendants.")

	return cmd
}
