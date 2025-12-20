package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/runtime"
)

// newDebugPromptCmd creates the debug prompt command
func newDebugPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug-prompt [branch]",
		Short: "Output the full prompt that will be passed to the AI for PR description generation",
		Long: `Output the full prompt that will be passed to the AI for PR description generation.
This command collects all context (commit messages, code diffs, branch relationships, etc.)
and formats it as the complete prompt that would be sent to the AI service.

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

			// Build and output the full prompt
			prompt := ai.BuildPrompt(prCtx)
			fmt.Println(prompt)
			return nil
		},
	}

	return cmd
}
