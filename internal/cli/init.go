package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// isInteractive checks if we're in an interactive terminal
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// inferTrunk attempts to infer the trunk branch name
// Exported so it can be used by other commands
func InferTrunk(branchNames []string) string {
	// First, try to find a remote branch (check origin)
	remoteBranch, err := git.FindRemoteBranch("origin")
	if err == nil && remoteBranch != "" {
		// Validate it exists in branch list
		for _, name := range branchNames {
			if name == remoteBranch {
				return remoteBranch
			}
		}
	}

	// Second, check for commonly named trunks
	if common := git.FindCommonlyNamedTrunk(branchNames); common != "" {
		return common
	}

	return ""
}

// selectTrunkBranch prompts user to select trunk branch (simplified for now)
func selectTrunkBranch(branchNames []string, inferredTrunk string, interactive bool) (string, error) {
	if !interactive {
		if inferredTrunk != "" {
			return inferredTrunk, nil
		}
		return "", fmt.Errorf("could not infer trunk branch, pass in an existing branch name with --trunk or run in interactive mode")
	}

	// For now, if we have an inferred trunk, use it
	// TODO: Add proper interactive prompt with bubbletea for full branch selection
	if inferredTrunk != "" {
		return inferredTrunk, nil
	}

	// Fallback: use first branch
	if len(branchNames) > 0 {
		return branchNames[0], nil
	}

	return "", fmt.Errorf("no branches available")
}

// newInitCmd creates the init command
func newInitCmd() *cobra.Command {
	var (
		trunk         string
		reset         bool
		noInteractive bool
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

			// Create splog for output
			splog := output.NewSplog()

			// Determine trunk
			trunkName := trunk
			if trunkName == "" {
				// Try to infer trunk
				inferredTrunk := InferTrunk(branchNames)
				
				// Select trunk (with interactive prompt if needed)
				interactive := !noInteractive && isInteractive()
				selected, err := selectTrunkBranch(branchNames, inferredTrunk, interactive)
				if err != nil {
					return err
				}
				trunkName = selected
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

			// Check if already initialized
			wasInitialized := config.IsInitialized(repoRoot)

			// Set trunk in config
			if err := config.SetTrunk(repoRoot, trunkName); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			// Output welcome message
			if wasInitialized {
				splog.Info("Reinitializing Stackit...")
			} else {
				splog.Info("Welcome to Stackit!")
			}
			splog.Newline()

			// Use output formatter for colored output
			coloredTrunk := output.ColorBranchName(trunkName, false)
			splog.Info("Trunk set to %s", coloredTrunk)

			// Create engine and perform reset/rebuild
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			if reset {
				if err := eng.Reset(trunkName); err != nil {
					return fmt.Errorf("failed to reset branches: %w", err)
				}
				splog.Info("All branches have been untracked")
			} else {
				if err := eng.Rebuild(trunkName); err != nil {
					return fmt.Errorf("failed to rebuild engine: %w", err)
				}
				splog.Info("Stackit initialized successfully!")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&trunk, "trunk", "", "The name of your trunk branch")
	cmd.Flags().BoolVar(&reset, "reset", false, "Untrack all branches")
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Disable interactive prompts")

	return cmd
}

