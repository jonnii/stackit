package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// isInteractive checks if we're in an interactive terminal
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// InferTrunk attempts to infer the trunk branch name
func InferTrunk(ctx context.Context, branchNames []string) string {
	remoteBranch, err := git.FindRemoteBranch(ctx, "origin")
	if err == nil && remoteBranch != "" {
		for _, name := range branchNames {
			if name == remoteBranch {
				return remoteBranch
			}
		}
	}

	if common := engine.FindCommonlyNamedTrunk(branchNames); common != "" {
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

	// TODO: Add proper interactive prompt with bubbletea for full branch selection
	if inferredTrunk != "" {
		return inferredTrunk, nil
	}

	if len(branchNames) > 0 {
		return branchNames[0], nil
	}

	return "", fmt.Errorf("no branches available")
}

// EnsureInitialized initializes stackit if not already initialized.
// Returns the repo root path. This is used by commands that need stackit
// to be initialized but want to auto-initialize for convenience.
func EnsureInitialized(ctx context.Context) (string, error) {
	if err := git.InitDefaultRepo(); err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repo root: %w", err)
	}

	cfg, _ := config.LoadConfig(repoRoot)
	if !cfg.IsInitialized() {
		splog := tui.NewSplog()
		splog.Info("Stackit has not been initialized, attempting to setup now...")

		branchNames, err := git.GetAllBranchNames()
		if err != nil {
			return "", fmt.Errorf("failed to get branches: %w", err)
		}

		if len(branchNames) == 0 {
			return "", fmt.Errorf("no branches found in current repo; cannot initialize Stackit.\nPlease create your first commit and then re-run stackit init")
		}

		trunkName := InferTrunk(ctx, branchNames)
		if trunkName == "" {
			trunkName = "main"
			found := false
			for _, name := range branchNames {
				if name == "main" {
					found = true
					break
				}
			}
			if !found && len(branchNames) > 0 {
				trunkName = branchNames[0]
			}
		}

		cfg.SetTrunk(trunkName)
		if err := cfg.Save(); err != nil {
			return "", fmt.Errorf("failed to initialize: %w", err)
		}
	}

	return repoRoot, nil
}

// newInitCmd creates the init command
func newInitCmd() *cobra.Command {
	var (
		trunk         string
		reset         bool
		noInteractive bool
	)

	cmd := &cobra.Command{
		Use:          "init",
		Aliases:      []string{"i"},
		Short:        "Initialize Stackit in the current repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			cfg, err := config.LoadConfig(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			branchNames, err := git.GetAllBranchNames()
			if err != nil {
				return fmt.Errorf("failed to get branches: %w", err)
			}

			if len(branchNames) == 0 {
				return fmt.Errorf("no branches found in current repo; cannot initialize Stackit.\nPlease create your first commit and then re-run stackit init")
			}

			splog := tui.NewSplog()

			trunkName := trunk
			if trunkName == "" {
				inferredTrunk := InferTrunk(cmd.Context(), branchNames)

				interactive := !noInteractive && isInteractive()
				selected, err := selectTrunkBranch(branchNames, inferredTrunk, interactive)
				if err != nil {
					return err
				}
				trunkName = selected
			} else {
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

			wasInitialized := cfg.IsInitialized()

			cfg.SetTrunk(trunkName)
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			if wasInitialized {
				splog.Info("Reinitializing Stackit...")
			} else {
				splog.Info("Welcome to Stackit!")
			}
			splog.Newline()

			coloredTrunk := style.ColorBranchName(trunkName, false)
			splog.Info("Trunk set to %s", coloredTrunk)

			maxUndoDepth := cfg.UndoStackDepth()

			eng, err := engine.NewEngine(engine.Options{
				RepoRoot:          repoRoot,
				Trunk:             trunkName,
				MaxUndoStackDepth: maxUndoDepth,
			})
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
