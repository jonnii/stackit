package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/runtime"
)

// newDebugContextCmd creates the debug context command
func newDebugContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug-context [branch]",
		Short: "Output all context collected for AI-powered PR description generation",
		Long: `Output all context collected for AI-powered PR description generation.
This command collects commit messages, code diffs, branch relationships, and project conventions
but does not perform any AI generation. Useful for debugging and testing context collection.

If no branch is specified, uses the current branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Determine branch name
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			} else {
				branchName = ctx.Engine.CurrentBranch()
				if branchName == "" {
					return fmt.Errorf("not on a branch and no branch specified")
				}
			}

			// Collect PR context
			prCtx, err := ai.CollectPRContext(ctx, ctx.Engine, branchName)
			if err != nil {
				return fmt.Errorf("failed to collect PR context: %w", err)
			}

			// Output as JSON for easy parsing
			output, err := json.MarshalIndent(prCtx, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal context: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	return cmd
}
