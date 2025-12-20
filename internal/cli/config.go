package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// newConfigCmd creates the config command
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set repository configuration",
		Long: `Get and set repository configuration values.

Examples:
  stackit config get branch-name-pattern
  stackit config set branch-name-pattern "{username}/{date}/{message}"
  stackit config get create.ai
  stackit config set create.ai true`,
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())

	return cmd
}

// newConfigGetCmd creates the config get command
func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repo root
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			key := args[0]
			switch key {
			case "branch-name-pattern":
				pattern, err := config.GetBranchNamePattern(repoRoot)
				if err != nil {
					return fmt.Errorf("failed to get branch-name-pattern: %w", err)
				}
				fmt.Println(pattern)
			case "create.ai":
				enabled, err := config.GetCreateAI(repoRoot)
				if err != nil {
					return fmt.Errorf("failed to get create.ai: %w", err)
				}
				fmt.Println(enabled)
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			return nil
		},
	}

	return cmd
}

// newConfigSetCmd creates the config set command
func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repo root
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			key := args[0]
			value := args[1]

			splog := tui.NewSplog()

			switch key {
			case "branch-name-pattern":
				if err := config.SetBranchNamePattern(repoRoot, value); err != nil {
					return fmt.Errorf("failed to set branch-name-pattern: %w", err)
				}
				splog.Info("Set branch-name-pattern to: %s", value)
			case "create.ai":
				enabled, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for create.ai: %s (must be 'true' or 'false')", value)
				}
				if err := config.SetCreateAI(repoRoot, enabled); err != nil {
					return fmt.Errorf("failed to set create.ai: %w", err)
				}
				splog.Info("Set create.ai to: %v", enabled)
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			return nil
		},
	}

	return cmd
}
