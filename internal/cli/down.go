package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
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

			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get current branch
			currentBranch := ctx.Engine.CurrentBranch()
			if currentBranch == "" {
				return errors.ErrNotOnBranch
			}

			// Check if on trunk
			currentBranchObj := ctx.Engine.GetBranch(currentBranch)
			if currentBranchObj.IsTrunk() {
				ctx.Splog.Info("Already at trunk (%s).", tui.ColorBranchName(currentBranch, true))
				return nil
			}

			// Traverse down the specified number of steps
			targetBranch := currentBranch
			for i := 0; i < steps; i++ {
				parent := ctx.Engine.GetParent(targetBranch)
				if parent == "" {
					// No parent found - branch is untracked or we've gone past trunk
					if i == 0 {
						ctx.Splog.Info("%s has no parent (untracked branch).", tui.ColorBranchName(currentBranch, true))
						return nil
					}
					// We moved some steps but can't go further
					ctx.Splog.Info("Stopped at %s (no further parent after %d step(s)).", tui.ColorBranchName(targetBranch, false), i)
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
			if err := git.CheckoutBranch(ctx.Context, targetBranch); err != nil {
				return fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
			}

			ctx.Splog.Info("Checked out %s.", tui.ColorBranchName(targetBranch, false))
			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse downstack.")

	return cmd
}
