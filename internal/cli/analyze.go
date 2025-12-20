package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/runtime"
)

func newAnalyzeCmd() *cobra.Command {
	var verbose int

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze staged changes and suggest a stack structure",
		Long:  "Uses AI to analyze your staged changes and suggest how to split them into a logical stack of branches.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			aiClient, err := ai.NewCursorAgentClient()
			if err != nil {
				return fmt.Errorf("AI features require cursor-agent CLI: %w", err)
			}

			opts := actions.AnalyzeOptions{
				AIClient: aiClient,
				Verbose:  verbose,
			}

			suggestion, err := actions.AnalyzeAction(ctx, opts)
			if err != nil {
				return err
			}

			// Display suggestion
			printSuggestion(suggestion)

			return nil
		},
	}

	cmd.Flags().CountVarP(&verbose, "verbose", "v", "verbose output")

	return cmd
}

func printSuggestion(suggestion *ai.StackSuggestion) {
	fmt.Println("\n🤖 Suggested stack structure:")
	fmt.Println()

	for i, layer := range suggestion.Layers {
		fmt.Printf("Branch %d: %q (%d files)\n", i+1, layer.BranchName, len(layer.Files))
		if len(layer.Files) > 0 {
			fmt.Printf("  - %s\n", strings.Join(layer.Files, ", "))
		}
		if layer.Rationale != "" {
			fmt.Printf("  - Rationale: %s\n", layer.Rationale)
		}
		if layer.CommitMessage != "" {
			fmt.Printf("  - Commit: %s\n", layer.CommitMessage)
		}
		fmt.Println()
	}
}
