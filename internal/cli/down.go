package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// newDownCmd creates the down command
func newDownCmd() *cobra.Command {
	var (
		steps int
	)

	cmd := &cobra.Command{
		Use:   "down [steps]",
		Short: "Switch to the parent of the current branch",
		Long: `Switch to the parent of the current branch.

Navigates down the stack toward trunk by switching to the parent branch.
By default, moves one level down. Use the --steps flag or pass a number
as an argument to move multiple levels at once.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse steps from positional argument if provided
			if len(args) > 0 {
				parsedSteps, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid steps argument: %s (must be a number)", args[0])
				}
				steps = parsedSteps
			}

			if steps < 1 {
				return fmt.Errorf("steps must be at least 1")
			}

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
				return errors.ErrNotOnBranch
			}

			// Check if on trunk
			if ctx.Engine.IsTrunk(currentBranch) {
				ctx.Splog.Info("Already at trunk (%s).", output.ColorBranchName(currentBranch, true))
				return nil
			}

			// Traverse down the specified number of steps
			targetBranch := currentBranch
			for i := 0; i < steps; i++ {
				parent := ctx.Engine.GetParent(targetBranch)
				if parent == "" {
					// No parent found - branch is untracked or we've gone past trunk
					if i == 0 {
						ctx.Splog.Info("%s has no parent (untracked branch).", output.ColorBranchName(currentBranch, true))
						return nil
					}
					// We moved some steps but can't go further
					ctx.Splog.Info("Stopped at %s (no further parent after %d step(s)).", output.ColorBranchName(targetBranch, false), i)
					break
				}
				ctx.Splog.Info("⮑  %s", parent)
				targetBranch = parent
			}

			// Check if we actually moved
			if targetBranch == currentBranch {
				ctx.Splog.Info("Already at the bottom of the stack.")
				return nil
			}

			// Checkout the target branch
			if err := git.CheckoutBranch(targetBranch); err != nil {
				return fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
			}

			ctx.Splog.Info("Checked out %s.", output.ColorBranchName(targetBranch, false))
			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse downstack.")

	return cmd
}
