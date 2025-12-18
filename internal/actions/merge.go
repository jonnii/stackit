package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// MergeOptions contains options for the merge command
type MergeOptions struct {
	DryRun   bool
	Confirm  bool
	Strategy MergeStrategy
	Force    bool
}

// MergeAction performs the merge operation using the plan/execute pattern
func MergeAction(ctx *runtime.Context, opts MergeOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Default strategy to bottom-up if not specified
	strategy := opts.Strategy
	if strategy == "" {
		strategy = MergeStrategyBottomUp
	}

	// 1. Populate remote SHAs so we can accurately check if branches match remote
	// This must be called before creating the merge plan so BranchMatchesRemote works correctly
	if err := eng.PopulateRemoteShas(ctx.Context); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
		// Continue anyway - we'll just have less accurate remote matching info
	} else {
		splog.Debug("Populated remote SHAs for branch matching")
	}

	// 2. Check sync status
	needsSync, staleBranches, err := CheckSyncStatus(ctx.Context, eng, splog)
	if err != nil {
		return fmt.Errorf("failed to check sync status: %w", err)
	}

	if needsSync {
		splog.Warn("Repository is not up to date with remote")
		if len(staleBranches) > 0 {
			splog.Info("Stale branches: %v", staleBranches)
		}
		splog.Tip("Run 'stackit sync' to update before merging")
		// In interactive mode, we would prompt here, but for now we'll continue
		// The plan creation will validate individual branches
	}

	// 3. Create merge plan
	plan, validation, err := CreateMergePlan(ctx, CreateMergePlanOptions{
		Strategy: strategy,
		Force:    opts.Force,
	})
	if err != nil {
		return err
	}

	// 4. Display validation errors if any
	if !validation.Valid {
		splog.Warn("Cannot proceed with merge due to validation errors:")
		for _, errMsg := range validation.Errors {
			splog.Warn("  ✗ %s", errMsg)
		}
		// In dry-run mode, show the plan anyway
		if !opts.DryRun && !opts.Force {
			return fmt.Errorf("validation failed (use --force to override)")
		}
		if !opts.DryRun && opts.Force {
			splog.Warn("Proceeding despite validation errors (--force enabled)")
		}
	}

	// 5. Display warnings if any
	if len(validation.Warnings) > 0 {
		splog.Warn("Warnings:")
		for _, warn := range validation.Warnings {
			splog.Warn("  ⚠ %s", warn)
		}
		splog.Newline()
		// Block execution if there are warnings (unless --force or dry-run)
		if !opts.DryRun && !opts.Force {
			splog.Warn("Cannot proceed with merge due to warnings. Use --force to override.")
			return fmt.Errorf("merge blocked due to warnings (use --force to override)")
		}
		if !opts.DryRun && opts.Force {
			splog.Warn("Proceeding despite warnings (--force enabled)")
		}
	}

	// 6. Display plan (dry-run or preview)
	planText := FormatMergePlan(plan, validation)
	splog.Page(planText)

	// If dry-run, stop here
	if opts.DryRun {
		return nil
	}

	// 6. Confirm if needed
	if opts.Confirm {
		confirmed, err := tui.PromptConfirm("Proceed with merge?", false)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			splog.Info("Merge canceled")
			return nil
		}
	}

	// 7. Execute the plan
	if err := ExecuteMergePlan(ctx, ExecuteMergePlanOptions{
		Plan:  plan,
		Force: opts.Force,
	}); err != nil {
		return fmt.Errorf("merge execution failed: %w", err)
	}

	splog.Info("Merge completed successfully")
	return nil
}
