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

// newSplitCmd creates the split command
func newSplitCmd() *cobra.Command {
	var (
		byCommit bool
		byHunk   bool
		byFile   []string
	)

	cmd := &cobra.Command{
		Use:     "split",
		Aliases: []string{"sp"},
		Short:   "Split the current branch into multiple branches",
		Long: `Split the current branch into multiple branches.

Has three forms: split --by-commit, split --by-hunk, and split --by-file <pathspecs>.
split --by-commit slices up the commit history, allowing you to select split points.
split --by-hunk interactively stages changes to create new single-commit branches.
split --by-file <pathspecs> extracts files matching the pathspecs into a new parent branch.
All forms must be run interactively except for --by-file which can run non-interactively.
split without options will prompt for a splitting strategy.`,
		// Disable default help flag to allow -h for --by-hunk
		DisableFlagParsing: false,
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

			// Determine split style - check all flag variants
			var style actions.SplitStyle
			if byCommit || cmd.Flags().Changed("commit") {
				style = actions.SplitStyleCommit
			} else if byHunk || cmd.Flags().Changed("hunk") {
				style = actions.SplitStyleHunk
			} else if len(byFile) > 0 || cmd.Flags().Changed("file") {
				// Get file pathspecs from either flag
				if cmd.Flags().Changed("file") {
					filePaths, _ := cmd.Flags().GetStringSlice("file")
					byFile = filePaths
				}
				style = actions.SplitStyleFile
			}
			// If style is empty, SplitAction will prompt

			// Run split action
			return actions.SplitAction(actions.SplitOptions{
				Style:     style,
				Pathspecs: byFile,
				Engine:    eng,
				Splog:     ctx.Splog,
				RepoRoot:  repoRoot,
			})
		},
	}

	// Disable the default help flag shorthand to allow -h for --by-hunk
	cmd.Flags().BoolP("help", "", false, "help for split")

	// Define flags - cobra allows multiple long forms but only one shorthand per variable
	cmd.Flags().BoolVarP(&byCommit, "by-commit", "c", false, "Split by commit - slice up the history of this branch")
	cmd.Flags().BoolVarP(&byHunk, "by-hunk", "h", false, "Split by hunk - split into new single-commit branches")
	cmd.Flags().StringSliceVarP(&byFile, "by-file", "f", nil, "Split by file - takes a number of pathspecs and splits any matching files into a new parent branch")

	// Add alternative long form names (these will be checked in RunE via cmd.Flags().Changed)
	// Note: We can't bind the same variable twice, so we check for these flags manually
	_ = cmd.Flags().Bool("commit", false, "Alias for --by-commit")
	_ = cmd.Flags().Bool("hunk", false, "Alias for --by-hunk")
	_ = cmd.Flags().StringSlice("file", nil, "Alias for --by-file")

	return cmd
}
