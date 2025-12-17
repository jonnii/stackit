package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// newTrunkCmd creates the trunk command
func newTrunkCmd() *cobra.Command {
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

			// Check if initialized
			if !config.IsInitialized(repoRoot) {
				return fmt.Errorf("stackit is not initialized. Run 'stackit init' first")
			}

			// Handle --add flag
			if add != "" {
				return handleAddTrunk(repoRoot, add)
			}

			// Handle --all flag
			if all {
				return handleShowAllTrunks(repoRoot)
			}

			// Default: show trunk for current branch
			return handleShowTrunk(repoRoot)
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
	if err := config.AddTrunk(repoRoot, trunkName); err != nil {
		return err
	}

	splog := tui.NewSplog()
	splog.Info("Added %s as a trunk branch.", tui.ColorBranchName(trunkName, false))
	return nil
}

// handleShowAllTrunks shows all configured trunk branches
func handleShowAllTrunks(repoRoot string) error {
	trunks, err := config.GetAllTrunks(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get trunks: %w", err)
	}

	// Get primary trunk to mark it
	primaryTrunk, _ := config.GetTrunk(repoRoot)

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
func handleShowTrunk(repoRoot string) error {
	// Create engine
	eng, err := engine.NewEngine(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Create context
	ctx := runtime.NewContext(eng)

	// Get current branch
	currentBranch := ctx.Engine.CurrentBranch()
	if currentBranch == "" {
		// Not on a branch, just show primary trunk
		trunk := ctx.Engine.Trunk()
		fmt.Println(trunk)
		return nil
	}

	// If current branch is trunk, show it
	if ctx.Engine.IsTrunk(currentBranch) {
		fmt.Println(currentBranch)
		return nil
	}

	// Find the trunk by walking up the parent chain
	trunk := findTrunkForBranch(ctx.Engine, currentBranch, repoRoot)
	fmt.Println(trunk)
	return nil
}

// findTrunkForBranch walks up the parent chain to find the trunk
func findTrunkForBranch(eng engine.Engine, branchName string, repoRoot string) string {
	// Get all configured trunks
	trunks, err := config.GetAllTrunks(repoRoot)
	if err != nil {
		return eng.Trunk()
	}

	// Walk up the parent chain
	current := branchName
	visited := make(map[string]bool)

	for current != "" && !visited[current] {
		visited[current] = true

		// Check if current is a trunk
		for _, t := range trunks {
			if current == t {
				return current
			}
		}

		// Get parent
		parent := eng.GetParent(current)
		if parent == "" {
			break
		}
		current = parent
	}

	// Default to primary trunk
	return eng.Trunk()
}
