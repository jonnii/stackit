package navigation

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
)

// NewDownCmd creates the down command
func NewDownCmd() *cobra.Command {
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
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *runtime.Context) error {
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

				// Get current branch
				currentBranch := ctx.Engine.CurrentBranch()
				if currentBranch == nil {
					return errors.ErrNotOnBranch
				}

				// Check if on trunk
				if currentBranch.IsTrunk() {
					ctx.Splog.Info("Already at trunk (%s).", style.ColorBranchName(currentBranch.GetName(), true))
					return nil
				}

				// Traverse down the specified number of steps
				targetBranch := *currentBranch
				for i := 0; i < steps; i++ {
					parent := ctx.Engine.GetParent(targetBranch)
					if parent == nil {
						// No parent found - branch is untracked or we've gone past trunk
						if i == 0 {
							ctx.Splog.Info("%s has no parent (untracked branch).", style.ColorBranchName(currentBranch.GetName(), true))
							return nil
						}
						// We moved some steps but can't go further
						ctx.Splog.Info("Stopped at %s (no further parent after %d step(s)).", style.ColorBranchName(targetBranch.GetName(), false), i)
						break
					}
					ctx.Splog.Info("⮑  %s", parent.GetName())
					targetBranch = *parent
				}

				// Check if we actually moved
				if targetBranch.GetName() == currentBranch.GetName() {
					ctx.Splog.Info("Already at the bottom of the stack.")
					return nil
				}

				// Checkout the target branch
				if err := ctx.Engine.CheckoutBranch(ctx.Context, targetBranch); err != nil {
					return fmt.Errorf("failed to checkout branch %s: %w", targetBranch.GetName(), err)
				}

				ctx.Splog.Info("Checked out %s.", style.ColorBranchName(targetBranch.GetName(), false))
				return nil
			})
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse downstack.")

	return cmd
}
