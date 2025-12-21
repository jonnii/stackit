package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/runtime"
)

// newSplitCmd creates the split command
func newSplitCmd() *cobra.Command {
	var (
		byCommit          bool
		byHunk            bool
		byFile            []string
		byFileInteractive bool
	)

	cmd := &cobra.Command{
		Use:     "split",
		Aliases: []string{"sp"},
		Short:   "Split the current branch into multiple branches",
		Long: `Split the current branch into multiple branches.

Has three forms: split --by-commit, split --by-hunk, and split --by-file.
split --by-commit slices up the commit history, allowing you to select split points.
split --by-hunk interactively stages changes to create new single-commit branches.
split --by-file <files> extracts specified files into a new parent branch.
split -F (--by-file-interactive) shows an interactive file selector.
split without options will prompt for a splitting strategy.`,
		// Disable default help flag to allow -h for --by-hunk
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Determine split style - check all flag variants
			var style actions.SplitStyle
			switch {
			case byCommit || cmd.Flags().Changed("commit"):
				style = actions.SplitStyleCommit
			case byHunk || cmd.Flags().Changed("hunk"):
				style = actions.SplitStyleHunk
			case byFileInteractive || len(byFile) > 0 || cmd.Flags().Changed("file"):
				// -F triggers interactive file selection
				// --by-file with pathspecs uses those files directly
				if cmd.Flags().Changed("file") {
					filePaths, _ := cmd.Flags().GetStringSlice("file")
					byFile = filePaths
				}
				style = actions.SplitStyleFile
			}
			// If style is empty, SplitAction will prompt

			// Run split action
			return actions.SplitAction(ctx, actions.SplitOptions{
				Style:     style,
				Pathspecs: byFile,
			})
		},
	}

	// Disable the default help flag shorthand to allow -h for --by-hunk
	cmd.Flags().BoolP("help", "", false, "help for split")

	// Define flags - cobra allows multiple long forms but only one shorthand per variable
	cmd.Flags().BoolVarP(&byCommit, "by-commit", "c", false, "Split by commit - slice up the history of this branch")
	cmd.Flags().BoolVarP(&byHunk, "by-hunk", "h", false, "Split by hunk - split into new single-commit branches")
	cmd.Flags().StringSliceVarP(&byFile, "by-file", "f", nil, "Split by file - extracts specified files to a new parent branch")
	cmd.Flags().BoolVarP(&byFileInteractive, "by-file-interactive", "F", false, "Split by file (interactive) - select files to extract")

	// Add alternative long form names (these will be checked in RunE via cmd.Flags().Changed)
	// Note: We can't bind the same variable twice, so we check for these flags manually
	_ = cmd.Flags().Bool("commit", false, "Alias for --by-commit")
	_ = cmd.Flags().Bool("hunk", false, "Alias for --by-hunk")
	_ = cmd.Flags().StringSlice("file", nil, "Alias for --by-file")

	return cmd
}
