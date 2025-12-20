package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// newConfigCmd creates the config command
func newConfigCmd() *cobra.Command {
	var listFlag bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set repository configuration",
		Long: `Get and set repository configuration values.

When run without subcommands, opens an interactive TUI for editing configuration.
Use --list to print all configuration values instead.

Examples:
  stackit config                    # Interactive TUI
  stackit config --list             # Print all config values
  stackit config get branch.pattern
  stackit config set branch.pattern "{username}/{date}/{message}"
  stackit config get create.ai
  stackit config set create.ai true
  stackit config get submit.footer
  stackit config set submit.footer false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repo root
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			// If --list flag is set, or terminal is not interactive, show list
			if listFlag || !tui.IsTTY() {
				return actions.ConfigListAction(repoRoot)
			}

			// Otherwise, show interactive TUI
			return actions.ConfigTUIAction(repoRoot)
		},
	}

	cmd.Flags().BoolVarP(&listFlag, "list", "l", false, "Print all configuration values instead of opening interactive TUI")

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
			case "branch.pattern":
				pattern, err := config.GetBranchNamePattern(repoRoot)
				if err != nil {
					return fmt.Errorf("failed to get branch.pattern: %w", err)
				}
				fmt.Println(pattern)
			case "create.ai":
				enabled, err := config.GetCreateAI(repoRoot)
				if err != nil {
					return fmt.Errorf("failed to get create.ai: %w", err)
				}
				fmt.Println(enabled)
			case "submit.footer":
				enabled, err := config.GetSubmitFooter(repoRoot)
				if err != nil {
					return fmt.Errorf("failed to get submit.footer: %w", err)
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
			case "branch.pattern":
				if err := config.SetBranchNamePattern(repoRoot, value); err != nil {
					return fmt.Errorf("failed to set branch.pattern: %w", err)
				}
				splog.Info("Set branch.pattern to: %s", value)
			case "create.ai":
				enabled, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for create.ai: %s (must be 'true' or 'false')", value)
				}
				if err := config.SetCreateAI(repoRoot, enabled); err != nil {
					return fmt.Errorf("failed to set create.ai: %w", err)
				}
				splog.Info("Set create.ai to: %v", enabled)
			case "submit.footer":
				enabled, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for submit.footer: %s (must be 'true' or 'false')", value)
				}
				if err := config.SetSubmitFooter(repoRoot, enabled); err != nil {
					return fmt.Errorf("failed to set submit.footer: %w", err)
				}
				splog.Info("Set submit.footer to: %v", enabled)
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			return nil
		},
	}

	return cmd
}
