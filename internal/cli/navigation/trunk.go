package navigation

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// NewTrunkCmd creates the trunk command
func NewTrunkCmd() *cobra.Command {
	var (
		add string
		all bool
	)

	cmd := &cobra.Command{
		Use:   "trunk",
		Short: "Show the trunk of the current branch",
		Long: `Show the trunk of the current branch.

By default, displays the trunk branch that the current branch's stack is based on.
Use --all to see all configured trunk branches, or --add to add an additional trunk.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Handle --add flag
			if add != "" {
				return handleAddTrunk(ctx.RepoRoot, add)
			}

			// Handle --all flag
			if all {
				return handleShowAllTrunks(ctx.RepoRoot)
			}

			// Default: show trunk for current branch
			return handleShowTrunk(ctx)
		},
	}

	cmd.Flags().StringVar(&add, "add", "", "Add an additional trunk branch")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all configured trunks")

	return cmd
}

// handleAddTrunk adds a new trunk branch
func handleAddTrunk(repoRoot string, trunkName string) error {
	// Verify the branch exists
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	found := false
	for _, b := range branches {
		if b == trunkName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("branch '%s' does not exist", trunkName)
	}

	// Add the trunk
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.AddTrunk(trunkName); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	splog := tui.NewSplog()
	splog.Info("Added %s as a trunk branch.", tui.ColorBranchName(trunkName, false))
	return nil
}

// handleShowAllTrunks shows all configured trunk branches
func handleShowAllTrunks(repoRoot string) error {
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	trunks := cfg.AllTrunks()

	// Get primary trunk to mark it
	primaryTrunk := cfg.Trunk()

	for _, trunk := range trunks {
		if trunk == primaryTrunk {
			fmt.Printf("%s (primary)\n", trunk)
		} else {
			fmt.Println(trunk)
		}
	}

	return nil
}

// handleShowTrunk shows the trunk for the current branch
func handleShowTrunk(ctx *runtime.Context) error {
	eng := ctx.Engine

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		// Not on a branch, just show primary trunk
		trunk := eng.Trunk()
		fmt.Println(trunk.Name)
		return nil
	}

	// If current branch is trunk, show it
	if currentBranch.IsTrunk() {
		fmt.Println(currentBranch.Name)
		return nil
	}

	// Find the trunk by walking up the parent chain
	trunk := findTrunkForBranch(eng, currentBranch.Name, ctx.RepoRoot)
	fmt.Println(trunk)
	return nil
}

// findTrunkForBranch walks up the parent chain to find the trunk
func findTrunkForBranch(eng engine.Engine, branchName string, repoRoot string) string {
	// Get all configured trunks
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return eng.Trunk().Name
	}
	trunks := cfg.AllTrunks()

	// Walk up the parent chain
	currentBranch := eng.GetBranch(branchName)
	visited := make(map[string]bool)

	for currentBranch.Name != "" && !visited[currentBranch.Name] {
		visited[currentBranch.Name] = true

		// Check if current is a trunk
		for _, t := range trunks {
			if currentBranch.Name == t {
				return currentBranch.Name
			}
		}

		// Get parent
		parent := eng.GetParent(currentBranch)
		if parent == nil {
			break
		}
		currentBranch = *parent
	}

	// Default to primary trunk
	return eng.Trunk().Name
}
