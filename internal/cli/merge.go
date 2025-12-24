package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// newMergeCmd creates the merge command
func newMergeCmd() *cobra.Command {
	var (
		dryRun   bool
		yes      bool
		force    bool
		strategy string
		worktree bool
		scope    string
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge the pull requests associated with all branches from trunk to the current branch via Stackit",
		Long: `Merge the pull requests associated with all branches from trunk to the current branch via Stackit.
This command merges PRs for all branches in the stack from trunk up to (and including) the current branch.

If --scope is specified, all branches with that scope will be merged.

If no flags are provided, an interactive wizard will guide you through the merge process.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get context (demo or real)
			ctx, err := runtime.GetContext(cmd.Context())
			if err != nil {
				return err
			}

			// Determine if we should run in interactive mode
			// Interactive if no flags are provided (except dry-run and scope which are always allowed)
			interactive := strategy == "" && !yes && !force

			// Parse strategy
			var mergeStrategy merge.Strategy
			if strategy != "" {
				switch strings.ToLower(strategy) {
				case "bottom-up", "bottomup":
					mergeStrategy = merge.StrategyBottomUp
				case "top-down", "topdown":
					mergeStrategy = merge.StrategyTopDown
				default:
					return fmt.Errorf("invalid strategy: %s (must be 'bottom-up' or 'top-down')", strategy)
				}
			}

			// Run interactive wizard if needed
			if interactive {
				return runInteractiveMergeWizard(ctx, dryRun, force, scope)
			}

			// Get config values
			undoStackDepth, _ := config.GetUndoStackDepth(ctx.RepoRoot)

			// Create plan if scope is specified
			var plan *merge.Plan
			if scope != "" {
				p, _, err := merge.CreateMergePlan(ctx.Context, ctx.Engine, ctx.Splog, ctx.GitHubClient, merge.CreatePlanOptions{
					Strategy: mergeStrategy,
					Force:    force,
					Scope:    scope,
				})
				if err != nil {
					return err
				}
				plan = p
			}

			// Run merge action
			return merge.Action(ctx, merge.Options{
				DryRun:         dryRun,
				Confirm:        !yes, // If --yes is set, don't confirm
				Strategy:       mergeStrategy,
				Force:          force,
				UseWorktree:    worktree,
				Plan:           plan,
				UndoStackDepth: undoStackDepth,
			})
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "", "Merge strategy: 'bottom-up' (merge each PR from bottom) or 'top-down' (squash into one PR). Interactive if not specified.")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")
	cmd.Flags().BoolVar(&worktree, "worktree", false, "Execute the merge and restack in a temporary worktree to avoid interfering with current branch")
	cmd.Flags().StringVar(&scope, "scope", "", "Bulk-merge all branches within the specified scope")

	return cmd
}

// runInteractiveMergeWizard runs the interactive merge wizard
func runInteractiveMergeWizard(ctx *runtime.Context, dryRun bool, forceFlag bool, scope string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	splog.Info("🔍 Analyzing stack...")
	splog.Newline()

	// Populate remote SHAs so we can accurately check if branches match remote
	if err := eng.PopulateRemoteShas(); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Get current branch info
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	// Create initial plan with bottom-up strategy (default)
	plan, validation, err := merge.CreateMergePlan(ctx.Context, eng, splog, ctx.GitHubClient, merge.CreatePlanOptions{
		Strategy: merge.StrategyBottomUp,
		Force:    forceFlag,
		Scope:    scope,
	})
	if err != nil {
		return err
	}

	// Display current state using stack tree
	if scope != "" {
		splog.Info("Merging scope: [%s]", scope)
	} else {
		splog.Info("You are on branch: %s", tui.ColorBranchName(currentBranch.Name, false))
	}
	splog.Newline()

	if len(plan.BranchesToMerge) > 0 {
		// Create tree renderer
		renderer := tui.NewStackTreeRenderer(
			currentBranch.Name,
			eng.Trunk().Name,
			func(branchName string) []string {
				branch := eng.GetBranch(branchName)
				children := branch.GetChildren()
				childNames := make([]string, len(children))
				for i, c := range children {
					childNames[i] = c.Name
				}
				return childNames
			},
			func(branchName string) string {
				branch := eng.GetBranch(branchName)
				parent := eng.GetParent(branch)
				if parent == nil {
					return ""
				}
				return parent.Name
			},
			func(branchName string) bool { return eng.GetBranch(branchName).IsTrunk() },
			func(branchName string) bool {
				return eng.GetBranch(branchName).IsBranchUpToDate()
			},
		)

		// Build annotations for branches to merge
		annotations := make(map[string]tui.BranchAnnotation)
		for _, branchInfo := range plan.BranchesToMerge {
			annotation := tui.BranchAnnotation{
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
				splog.Info("  • %s", tui.ColorBranchName(branchName, false))
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
			splog.Warn("  %s", warn)
		}
		splog.Newline()
		if !forceFlag {
			splog.Info("Cannot proceed with merge due to warnings. Use --force to override validation checks.")
			return fmt.Errorf("merge blocked due to warnings (use --force to override)")
		}
		splog.Info("Proceeding despite warnings (--force enabled)")
	}

	// Show informational messages if any
	if len(validation.Infos) > 0 {
		splog.Info("Information:")
		for _, info := range validation.Infos {
			splog.Info("  • %s", info)
		}
		splog.Newline()
	}

	// Determine merge strategy
	var mergeStrategy merge.Strategy
	// If only a single PR, automatically use top-down strategy
	if len(plan.BranchesToMerge) == 1 {
		mergeStrategy = merge.StrategyTopDown
		splog.Info("✅ Strategy: %s (auto-selected for single PR)", mergeStrategy)
		splog.Newline()
	} else {
		// Prompt for strategy using interactive selector
		strategyOptions := []tui.SelectOption{
			{Label: "🔄 Bottom-up — Merge PRs one at a time from bottom (recommended)", Value: "bottom-up"},
			{Label: "📦 Top-down — Squash all changes into one PR, merge once", Value: "top-down"},
		}

		selectedStrategy, err := tui.PromptSelect("Select merge strategy:", strategyOptions, 0)
		if err != nil {
			return fmt.Errorf("strategy selection canceled: %w", err)
		}

		if selectedStrategy == "bottom-up" {
			mergeStrategy = merge.StrategyBottomUp
		} else {
			mergeStrategy = merge.StrategyTopDown
		}

		splog.Info("✅ Strategy: %s", mergeStrategy)
		splog.Newline()
	}

	// Recreate plan with selected strategy
	plan, validation, err = merge.CreateMergePlan(ctx.Context, eng, splog, ctx.GitHubClient, merge.CreatePlanOptions{
		Strategy: mergeStrategy,
		Force:    forceFlag,
	})
	if err != nil {
		return err
	}

	// Re-validate if strategy changed (important for top-down)
	if !validation.Valid && !forceFlag {
		splog.Warn("Errors found with selected strategy:")
		for _, errMsg := range validation.Errors {
			splog.Warn("  ✗ %s", errMsg)
		}
		return fmt.Errorf("validation failed")
	}

	// If dry-run, stop here
	if dryRun {
		splog.Info("📋 Merge Plan:")
		for i, step := range plan.Steps {
			splog.Info("  %d. %s", i+1, step.Description)
		}
		splog.Newline()
		splog.Info("Dry-run mode: plan displayed above. Use without --dry-run to execute.")
		return nil
	}

	// Prompt for confirmation
	confirmed, err := tui.PromptConfirm("Proceed with merge?", false)
	if err != nil {
		return fmt.Errorf("confirmation canceled: %w", err)
	}
	if !confirmed {
		splog.Info("Merge canceled")
		return nil
	}

	// Ask about worktree if not specified by flag
	useWorktree, err := tui.PromptConfirm("Execute merge in a temporary worktree? (allows you to continue working here)", true)
	if err != nil {
		return fmt.Errorf("worktree confirmation canceled: %w", err)
	}

	// Get config values
	undoStackDepth, _ := config.GetUndoStackDepth(ctx.RepoRoot)

	// Execute the plan
	mergeOpts := merge.Options{
		DryRun:         dryRun,
		Confirm:        false, // Already confirmed
		Strategy:       mergeStrategy,
		Force:          forceFlag,
		UseWorktree:    useWorktree,
		Plan:           plan,
		UndoStackDepth: undoStackDepth,
	}

	if err := merge.Action(ctx, mergeOpts); err != nil {
		return fmt.Errorf("merge action failed: %w", err)
	}

	return nil
}
