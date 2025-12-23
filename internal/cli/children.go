package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// newChildrenCmd creates the children command
func newChildrenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "children",
		Short: "Show the children of the current branch",
		Long: `Show the children of the current branch.

Lists all branches that have the current branch as their parent in the stack.
This is useful for understanding the structure of your stack and seeing which
branches depend on the current branch.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get current branch
			currentBranch := ctx.Engine.CurrentBranch()
			if currentBranch.Name == "" {
				return errors.ErrNotOnBranch
			}

			// Get children
			children := ctx.Engine.GetChildren(currentBranch.Name)
			if len(children) == 0 {
				ctx.Splog.Info("%s has no children.", tui.ColorBranchName(currentBranch.Name, true))
				return nil
			}

			// Print children
			for _, child := range children {
				fmt.Println(child)
			}
			return nil
		},
	}

	return cmd
}
