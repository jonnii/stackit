package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/demo"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// newMergeCmd creates the merge command
func newMergeCmd() *cobra.Command {
	var (
		dryRun   bool
		yes      bool
		force    bool
		strategy string
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge the pull requests associated with all branches from trunk to the current branch via Stackit",
		Long: `Merge the pull requests associated with all branches from trunk to the current branch via Stackit.
This command merges PRs for all branches in the stack from trunk up to (and including) the current branch.

If no flags are provided, an interactive wizard will guide you through the merge process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check for demo mode
			if ctx, ok := demo.NewDemoContext(); ok {
				// In demo mode, determine if interactive or non-interactive
				interactive := strategy == "" && !yes && !force

				// Parse strategy for non-interactive mode
				var mergeStrategy actions.MergeStrategy
				if strategy != "" {
					switch strings.ToLower(strategy) {
					case "bottom-up", "bottomup":
						mergeStrategy = actions.MergeStrategyBottomUp
					case "top-down", "topdown":
						mergeStrategy = actions.MergeStrategyTopDown
					default:
						return fmt.Errorf("invalid strategy: %s (must be 'bottom-up' or 'top-down')", strategy)
					}
				}

				if interactive {
					return runInteractiveMergeWizard(ctx.Engine, ctx, "", dryRun, true, force)
				}

				// Non-interactive demo mode
				return actions.MergeAction(actions.MergeOptions{
					DryRun:   dryRun,
					Confirm:  !yes,
					Strategy: mergeStrategy,
					Force:    force,
					Engine:   ctx.Engine,
					Splog:    ctx.Splog,
					RepoRoot: "",
					DemoMode: true,
				})
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
				return fmt.Errorf("stackit not initialized. Run 'stackit init' first")
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := runtime.NewContext(eng)

			// Determine if we should run in interactive mode
			// Interactive if no flags are provided (except dry-run which is always allowed)
			interactive := strategy == "" && !yes && !force

			// Parse strategy
			var mergeStrategy actions.MergeStrategy
			if strategy != "" {
				switch strings.ToLower(strategy) {
				case "bottom-up", "bottomup":
					mergeStrategy = actions.MergeStrategyBottomUp
				case "top-down", "topdown":
					mergeStrategy = actions.MergeStrategyTopDown
				default:
					return fmt.Errorf("invalid strategy: %s (must be 'bottom-up' or 'top-down')", strategy)
				}
			}

			// Run interactive wizard if needed
			if interactive {
				return runInteractiveMergeWizard(eng, ctx, repoRoot, dryRun, false, false)
			}

			// Run merge action
			return actions.MergeAction(actions.MergeOptions{
				DryRun:   dryRun,
				Confirm:  !yes, // If --yes is set, don't confirm
				Strategy: mergeStrategy,
				Force:    force,
				Engine:   eng,
				Splog:    ctx.Splog,
				RepoRoot: repoRoot,
			})
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "", "Merge strategy: 'bottom-up' (merge each PR from bottom) or 'top-down' (squash into one PR). Interactive if not specified.")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")

	return cmd
}

// runInteractiveMergeWizard runs the interactive merge wizard
func runInteractiveMergeWizard(eng engine.Engine, ctx *runtime.Context, repoRoot string, dryRun bool, demoMode bool, forceFlag bool) error {
	splog := ctx.Splog

	splog.Info("🔍 Analyzing stack...")
	splog.Newline()

	// Populate remote SHAs so we can accurately check if branches match remote
	if err := eng.PopulateRemoteShas(); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Get current branch info
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
		return fmt.Errorf("not on a branch")
	}

	// Create initial plan with bottom-up strategy (default)
	plan, validation, err := actions.CreateMergePlan(actions.CreateMergePlanOptions{
		Strategy: actions.MergeStrategyBottomUp,
		Force:    forceFlag,
		Engine:   eng,
		Splog:    splog,
		RepoRoot: repoRoot,
	})
	if err != nil {
		return err
	}

	// Display current state using stack tree
	splog.Info("You are on branch: %s", output.ColorBranchName(currentBranch, false))
	splog.Newline()

	if len(plan.BranchesToMerge) > 0 {
		// Create tree renderer
		renderer := output.NewStackTreeRenderer(
			currentBranch,
			eng.Trunk(),
			eng.GetChildren,
			eng.GetParent,
			eng.IsTrunk,
			eng.IsBranchFixed,
		)

		// Build annotations for branches to merge
		annotations := make(map[string]output.BranchAnnotation)
		for _, branchInfo := range plan.BranchesToMerge {
			annotation := output.BranchAnnotation{
				PRNumber:    &branchInfo.PRNumber,
				CheckStatus: string(branchInfo.ChecksStatus),
				IsDraft:     branchInfo.IsDraft,
			}
			annotations[branchInfo.BranchName] = annotation
		}
		renderer.SetAnnotations(annotations)

		// Render a list of branches to merge
		splog.Info("Stack to merge (bottom to top):")
		branchNames := make([]string, len(plan.BranchesToMerge))
		for i, branchInfo := range plan.BranchesToMerge {
			branchNames[i] = branchInfo.BranchName
		}
		stackLines := renderer.RenderBranchList(branchNames)
		for _, line := range stackLines {
			splog.Info("%s", line)
		}
		splog.Newline()

		// Show upstack branches that will be restacked
		if len(plan.UpstackBranches) > 0 {
			splog.Info("Branches above (will be restacked on trunk):")
			for _, branchName := range plan.UpstackBranches {
				splog.Info("  • %s", output.ColorBranchName(branchName, false))
			}
			splog.Newline()
		}
	}

	// Show validation errors if any
	if !validation.Valid {
		splog.Warn("Errors found:")
		for _, errMsg := range validation.Errors {
			splog.Warn("  ✗ %s", errMsg)
		}
		splog.Newline()
		splog.Info("Cannot proceed with merge. Use --force to override validation checks.")
		return fmt.Errorf("validation failed")
	}

	// Show warnings if any
	if len(validation.Warnings) > 0 {
		splog.Warn("Warnings:")
		for _, warn := range validation.Warnings {
			splog.Warn("  ⚠ %s", warn)
		}
		splog.Newline()
		if !forceFlag {
			splog.Info("Cannot proceed with merge due to warnings. Use --force to override validation checks.")
			return fmt.Errorf("merge blocked due to warnings (use --force to override)")
		}
		splog.Info("Proceeding despite warnings (--force enabled)")
	}

	// Prompt for strategy using interactive selector
	strategyOptions := []actions.SelectOption{
		{Label: "🔄 Bottom-up — Merge PRs one at a time from bottom (recommended)", Value: "bottom-up"},
		{Label: "📦 Top-down — Squash all changes into one PR, merge once", Value: "top-down"},
	}

	selectedStrategy, err := actions.PromptSelect("Select merge strategy:", strategyOptions, 0)
	if err != nil {
		return fmt.Errorf("strategy selection cancelled: %w", err)
	}

	var mergeStrategy actions.MergeStrategy
	if selectedStrategy == "bottom-up" {
		mergeStrategy = actions.MergeStrategyBottomUp
	} else {
		mergeStrategy = actions.MergeStrategyTopDown
	}

	splog.Info("✅ Strategy: %s", mergeStrategy)
	splog.Newline()

	// Recreate plan with selected strategy
	plan, validation, err = actions.CreateMergePlan(actions.CreateMergePlanOptions{
		Strategy: mergeStrategy,
		Force:    forceFlag,
		Engine:   eng,
		Splog:    splog,
		RepoRoot: repoRoot,
	})
	if err != nil {
		return err
	}

	// Display plan
	splog.Newline()
	splog.Info("📋 Merge Plan:")
	for i, step := range plan.Steps {
		splog.Info("  %d. %s", i+1, step.Description)
	}
	splog.Newline()

	// If dry-run, stop here
	if dryRun {
		splog.Info("Dry-run mode: plan displayed above. Use without --dry-run to execute.")
		return nil
	}

	// Prompt for confirmation
	confirmed, err := actions.PromptConfirm("Proceed with merge?", false)
	if err != nil {
		return fmt.Errorf("confirmation cancelled: %w", err)
	}
	if !confirmed {
		splog.Info("Merge cancelled")
		return nil
	}

	// Execute the plan
	if err := actions.ExecuteMergePlan(actions.ExecuteMergePlanOptions{
		Plan:     plan,
		Engine:   eng,
		Splog:    splog,
		RepoRoot: repoRoot,
		Force:    forceFlag,
		DemoMode: demoMode,
	}); err != nil {
		return fmt.Errorf("merge execution failed: %w", err)
	}

	splog.Info("Merge completed successfully")
	return nil
}
