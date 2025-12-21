package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// newParentCmd creates the parent command
func newParentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parent",
		Short: "Show the parent of the current branch",
		Long: `Show the parent of the current branch.

Displays the name of the branch that is the parent of the current branch
in the stack. This is useful for understanding the structure of your stack
and seeing which branch the current branch is based on.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			if ctx.Engine.IsTrunk(currentBranch) {
				ctx.Splog.Info("%s is trunk and has no parent.", tui.ColorBranchName(currentBranch, true))
				return nil
			}

			// Get parent
			parent := ctx.Engine.GetParent(currentBranch)
			if parent == "" {
				ctx.Splog.Info("%s has no parent (untracked branch).", tui.ColorBranchName(currentBranch, true))
				return nil
			}

			// Print parent
			fmt.Println(parent)
			return nil
		},
	}

	return cmd
}
