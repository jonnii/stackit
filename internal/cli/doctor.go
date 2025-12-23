package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/runtime"
)

// newDoctorCmd creates the doctor command
func newDoctorCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common issues with your stackit setup",
		Long: `Run diagnostic checks on your stackit environment and repository.

The doctor command checks:
  - Environment: Git version, GitHub CLI, and authentication
  - Repository: Git repository status, remote configuration, and trunk branch
  - Stack State: Metadata integrity, cycle detection, and missing parent branches`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Get config values
			trunk, _ := config.GetTrunk(ctx.RepoRoot)

			// Run doctor action
			return actions.DoctorAction(ctx, actions.DoctorOptions{
				Fix:   fix,
				Trunk: trunk,
			})
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to automatically fix any issues found")

	return cmd
}
