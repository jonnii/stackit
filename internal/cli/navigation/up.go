package navigation

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/cli/helpers"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// NewUpCmd creates the up command
func NewUpCmd() *cobra.Command {
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
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return helpers.Run(cmd, func(ctx *runtime.Context) error {
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

				// Traverse up the specified number of steps
				targetBranch := currentBranch.Name
				for i := 0; i < steps; i++ {
					targetBranchObj := ctx.Engine.GetBranch(targetBranch)
					children := targetBranchObj.GetChildren()
					if len(children) == 0 {
						if i == 0 {
							ctx.Splog.Info("Already at the top of the stack.")
							return nil
						}
						ctx.Splog.Info("Stopped at %s (no further children after %d step(s)).", style.ColorBranchName(targetBranch, false), i)
						break
					}

					var nextBranch string
					var err error
					if len(children) == 1 {
						nextBranch = children[0].Name
					} else {
						// Multiple children, decide which way to go
						if toBranch != "" {
							// Try to find the child that leads to toBranch
							var candidates []string
							for _, child := range children {
								upstack := ctx.Engine.GetRelativeStackUpstack(child)
								upstackNames := make([]string, len(upstack))
								for i, b := range upstack {
									upstackNames[i] = b.Name
								}
								if child.Name == toBranch || slices.Contains(upstackNames, toBranch) {
									candidates = append(candidates, child.Name)
								}
							}

							switch len(candidates) {
							case 1:
								nextBranch = candidates[0]
							case 0:
								// --to is not a descendant of any child
								ctx.Splog.Warn("Branch %s is not a descendant of %s.", style.ColorBranchName(toBranch, false), style.ColorBranchName(targetBranch, false))
								fallthrough
							default:
								// Still ambiguous even with --to (shouldn't happen in a tree)
								childNames := make([]string, len(children))
								for i, c := range children {
									childNames[i] = c.Name
								}
								nextBranch, err = promptForChild(childNames, targetBranch)
								if err != nil {
									return err
								}
							}
						} else {
							childNames := make([]string, len(children))
							for i, c := range children {
								childNames[i] = c.Name
							}
							nextBranch, err = promptForChild(childNames, targetBranch)
							if err != nil {
								return err
							}
						}
					}

					ctx.Splog.Info("⮑  %s", nextBranch)
					targetBranch = nextBranch
				}

				// Check if we actually moved
				if targetBranch == currentBranch.Name {
					return nil
				}

				// Checkout the target branch
				targetBranchObj := ctx.Engine.GetBranch(targetBranch)
				if err := git.CheckoutBranch(ctx.Context, targetBranchObj.Name); err != nil {
					return fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
				}

				ctx.Splog.Info("Checked out %s.", style.ColorBranchName(targetBranch, false))
				return nil
			})
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "The number of levels to traverse upstack.")
	cmd.Flags().StringVar(&toBranch, "to", "", "Target branch to navigate towards. When multiple children exist, selects the path leading to this branch.")

	_ = cmd.RegisterFlagCompletionFunc("to", helpers.CompleteBranches)

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
