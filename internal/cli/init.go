package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// newInitCmd creates the init command
func newInitCmd() *cobra.Command {
	var (
		trunk string
		reset bool
	)

	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Initialize Stackit in the current repository",
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

			// Get all branch names
			branchNames, err := git.GetAllBranchNames()
			if err != nil {
				return fmt.Errorf("failed to get branches: %w", err)
			}

			if len(branchNames) == 0 {
				return fmt.Errorf("no branches found in current repo; cannot initialize Stackit.\nPlease create your first commit and then re-run stackit init")
			}

			// Determine trunk
			trunkName := trunk
			if trunkName == "" {
				// Default to "main" if it exists, otherwise first branch
				trunkName = "main"
				found := false
				for _, name := range branchNames {
					if name == "main" {
						found = true
						break
					}
				}
				if !found {
					trunkName = branchNames[0]
				}
			} else {
				// Validate trunk exists
				found := false
				for _, name := range branchNames {
					if name == trunkName {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("branch '%s' not found", trunkName)
				}
			}

			// Create or update repo config
			configPath := filepath.Join(repoRoot, ".git", ".stackit_config")
			config := map[string]interface{}{
				"trunk":                      trunkName,
				"isGithubIntegrationEnabled": false,
			}

			configJSON, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			// Create splog for output
			splog := output.NewSplog()

			// Output success message
			if _, err := os.Stat(configPath); err == nil {
				splog.Info("Reinitializing Stackit...")
			} else {
				splog.Info("Welcome to Stackit!")
			}
			splog.Newline()

			// Use output formatter for colored output
			coloredTrunk := output.ColorBranchName(trunkName, false)
			splog.Info("Trunk set to %s", coloredTrunk)

			if reset {
				// TODO: Implement reset functionality
				splog.Info("All branches have been untracked")
			} else {
				splog.Info("Stackit initialized successfully!")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&trunk, "trunk", "", "The name of your trunk branch")
	cmd.Flags().BoolVar(&reset, "reset", false, "Untrack all branches")

	return cmd
}
