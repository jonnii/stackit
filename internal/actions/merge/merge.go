package merge

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// Options contains options for the merge command
type Options struct {
	DryRun         bool
	Confirm        bool
	Strategy       Strategy
	Force          bool
	UseWorktree    bool
	Plan           *Plan // Optional pre-calculated plan
	UndoStackDepth int   // Maximum undo stack depth (from config)
}

// Action performs the merge operation using the plan/execute pattern
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	plan := opts.Plan
	var validation *PlanValidation

	if plan == nil {
		// Default strategy to bottom-up if not specified
		strategy := opts.Strategy
		if strategy == "" {
			strategy = StrategyBottomUp
		}

		// 1. Populate remote SHAs so we can accurately check if branches match remote
		if err := eng.PopulateRemoteShas(); err != nil {
			splog.Debug("Failed to populate remote SHAs: %v", err)
		}

		// 2. Check sync status
		needsSync, staleBranches, err := CheckSyncStatus(ctx.Context, eng, splog)
		if err == nil && needsSync {
			splog.Warn("Repository is not up to date with remote")
			if len(staleBranches) > 0 {
				splog.Info("Stale branches: %v", staleBranches)
			}
			splog.Tip("Run 'stackit sync' to update before merging")
		}

		// 3. Create merge plan
		plan, validation, err = CreateMergePlan(ctx.Context, eng, splog, ctx.GitHubClient, CreatePlanOptions{
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
			if !opts.DryRun && !opts.Force {
				return fmt.Errorf("validation failed (use --force to override)")
			}
		}

		// 5. Display warnings if any
		if len(validation.Warnings) > 0 {
			splog.Warn("Warnings:")
			for _, warn := range validation.Warnings {
				splog.Warn("  %s", warn)
			}
			splog.Newline()
			if !opts.DryRun && !opts.Force {
				return fmt.Errorf("merge blocked due to warnings (use --force to override)")
			}
		}

		// Show informational messages if any
		if len(validation.Infos) > 0 {
			splog.Info("Information:")
			for _, info := range validation.Infos {
				splog.Info("  • %s", info)
			}
			splog.Newline()
		}

		// 6. Display plan (dry-run or preview)
		planText := FormatMergePlan(plan, validation)
		splog.Page(planText)
	}

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
	executeOpts := ExecuteOptions{
		Plan:           plan,
		Force:          opts.Force,
		UndoStackDepth: opts.UndoStackDepth,
	}

	if opts.UseWorktree {
		if err := ExecuteInWorktree(ctx.Context, eng, splog, ctx.GitHubClient, ctx.RepoRoot, executeOpts); err != nil {
			return fmt.Errorf("merge execution in worktree failed: %w", err)
		}
	} else {
		if err := Execute(ctx.Context, eng, splog, ctx.GitHubClient, ctx.RepoRoot, executeOpts); err != nil {
			return fmt.Errorf("merge execution failed: %w", err)
		}
	}

	splog.Info("Merge completed successfully")
	return nil
}
