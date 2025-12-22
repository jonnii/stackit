package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// newUpCmd creates the up command
func newUpCmd() *cobra.Command {
	var (
		steps    int
		toBranch string
	)

	cmd := &cobra.Command{
		Use:   "up [steps]",
		Short: "Switch to the child of the current branch",
		Long: `Switch to the child of the current branch.

Navigates up the stack away from trunk by switching to a child branch.
By default, moves one level up. Use the --steps flag or pass a number
as an argument to move multiple levels at once.

If multiple children exist, you will be prompted to select one, unless
the --to flag is used to specify a target branch to navigate towards.`,
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

			// Get context
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get current branch
			currentBranch := ctx.Engine.CurrentBranch()
			if currentBranch == "" {
				return errors.ErrNotOnBranch
			}

			// Traverse up the specified number of steps
			targetBranch := currentBranch
			for i := 0; i < steps; i++ {
				children := ctx.Engine.GetChildren(targetBranch)
				if len(children) == 0 {
					if i == 0 {
						ctx.Splog.Info("Already at the top of the stack.")
						return nil
					}
					ctx.Splog.Info("Stopped at %s (no further children after %d step(s)).", tui.ColorBranchName(targetBranch, false), i)
					break
				}

				var nextBranch string
				if len(children) == 1 {
					nextBranch = children[0]
				} else {
					// Multiple children, decide which way to go
					if toBranch != "" {
						// Try to find the child that leads to toBranch
						var candidates []string
						for _, child := range children {
							if child == toBranch || utils.ContainsString(ctx.Engine.GetRelativeStackUpstack(child), toBranch) {
								candidates = append(candidates, child)
							}
						}

						switch len(candidates) {
						case 1:
							nextBranch = candidates[0]
						case 0:
							// --to is not a descendant of any child
							ctx.Splog.Warn("Branch %s is not a descendant of %s.", tui.ColorBranchName(toBranch, false), tui.ColorBranchName(targetBranch, false))
							fallthrough
						default:
							// Still ambiguous even with --to (shouldn't happen in a tree)
							nextBranch, err = promptForChild(children, targetBranch)
							if err != nil {
								return err
							}
						}
					} else {
						nextBranch, err = promptForChild(children, targetBranch)
						if err != nil {
							return err
						}
					}
				}

				ctx.Splog.Info("⮑  %s", nextBranch)
				targetBranch = nextBranch
			}

			// Check if we actually moved
			if targetBranch == currentBranch {
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
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse upstack.")
	cmd.Flags().StringVar(&toBranch, "to", "", "Target branch to navigate towards. When multiple children exist, selects the path leading to this branch.")

	_ = cmd.RegisterFlagCompletionFunc("to", completeBranches)

	return cmd
}

func promptForChild(children []string, parent string) (string, error) {
	if !utils.IsInteractive() {
		return "", fmt.Errorf("multiple children found for %s; use --to or move in interactive mode", parent)
	}

	options := make([]tui.SelectOption, len(children))
	for i, child := range children {
		options[i] = tui.SelectOption{
			Label: child,
			Value: child,
		}
	}

	selected, err := tui.PromptSelect(fmt.Sprintf("Multiple children found for %s. Select one to move up:", parent), options, 0)
	if err != nil {
		return "", err
	}

	return selected, nil
}
