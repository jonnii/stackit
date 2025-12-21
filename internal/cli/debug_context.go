package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// newDebugPromptCmd creates the debug prompt command
func newDebugPromptCmd() *cobra.Command {
	var aiFlag bool

	cmd := &cobra.Command{
		Use:   "debug-prompt [branch]",
		Short: "Output the full prompt that will be passed to the AI for PR description generation",
		Long: `Output the full prompt that will be passed to the AI for PR description generation.
This command collects all context (commit messages, code diffs, branch relationships, etc.)
and formats it as the complete prompt that would be sent to the AI service.

If --ai is specified, also generates and displays what the commit message would be
if running 'stackit create --ai' with the current staged changes.

If no branch is specified, uses the current branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// If --ai flag is set, generate commit message from staged changes
			if aiFlag {
				// Check for staged changes
				hasStaged, err := git.HasStagedChanges(ctx.Context)
				if err != nil {
					return fmt.Errorf("failed to check staged changes: %w", err)
				}

				hasUnstaged, err := git.HasUnstagedChanges(ctx.Context)
				if err != nil {
					return fmt.Errorf("failed to check unstaged changes: %w", err)
				}

				hasUntracked, err := git.HasUntrackedFiles(ctx.Context)
				if err != nil {
					return fmt.Errorf("failed to check untracked files: %w", err)
				}

				if !hasStaged && !hasUnstaged && !hasUntracked {
					return fmt.Errorf("no changes available. Stage some changes first to generate a commit message")
				}

				// Get staged diff (or stage changes if needed)
				var diff string
				if hasStaged {
					diff, err = git.GetStagedDiff(ctx.Context)
					if err != nil {
						return fmt.Errorf("failed to get staged diff: %w", err)
					}
				} else if hasUnstaged || hasUntracked {
					// For debug purposes, we'll show what would be staged
					// But we won't actually stage it
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Note: No staged changes found. Showing what would be generated from unstaged/untracked files.\n\n")
					// Get unstaged diff as preview
					if hasUnstaged {
						diff, err = git.GetUnstagedDiff(ctx.Context)
						if err != nil {
							return fmt.Errorf("failed to get unstaged diff: %w", err)
						}
					}
				}

				if diff == "" {
					return fmt.Errorf("no diff available to generate commit message")
				}

				// Create AI client
				var aiClient ai.Client
				cursorClient, err := ai.NewCursorAgentClient()
				if err != nil {
					// Fall back to mock for demonstration
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Warning: cursor-agent not available, using mock client for demonstration\n\n")
					mockClient := ai.NewMockClient()
					mockClient.SetMockCommitMessage("feat: example commit message (mock)")
					aiClient = mockClient
				} else {
					aiClient = cursorClient
				}

				// Generate commit message
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "=== AI-Generated Commit Message ===\n\n")
				commitMessage, err := aiClient.GenerateCommitMessage(ctx.Context, diff)
				if err != nil {
					// Check if it's a cursor-agent error and provide helpful message
					errMsg := err.Error()
					if strings.Contains(errMsg, "cursor-agent failed") || strings.Contains(errMsg, "resource_exhausted") {
						return fmt.Errorf("cursor-agent failed: %w\n\nThis might be due to:\n- Rate limiting or resource exhaustion\n- Network connectivity issues\n- Invalid API credentials\n\nTry again later or check your cursor-agent configuration", err)
					}
					return fmt.Errorf("failed to generate commit message: %w", err)
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", commitMessage)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "=== Commit Message Prompt ===\n\n")
				prompt := ai.BuildCommitMessagePrompt(diff)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", prompt)
				return nil
			}

			// Original behavior: show PR description prompt
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

	cmd.Flags().BoolVar(&aiFlag, "ai", false, "Generate commit message from staged changes (as would be done with 'stackit create --ai')")

	return cmd
}
