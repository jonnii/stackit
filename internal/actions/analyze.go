package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// AnalyzeOptions contains options for the analyze command
type AnalyzeOptions struct {
	AIClient ai.Client
	Verbose  int
}

// AnalyzeAction analyzes staged changes and suggests a stack structure
func AnalyzeAction(ctx *runtime.Context, opts AnalyzeOptions) (*ai.StackSuggestion, error) {
	splog := ctx.Splog

	// Check if AI client is available
	if opts.AIClient == nil {
		return nil, fmt.Errorf("AI client not available")
	}

	// Handle staging if needed
	hasStaged, err := git.HasStagedChanges(ctx.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !hasStaged {
		hasUntracked, _ := git.HasUntrackedFiles(ctx.Context)
		hasUnstaged, _ := git.HasUnstagedChanges(ctx.Context)
		if !hasUntracked && !hasUnstaged {
			return nil, fmt.Errorf("no changes to analyze. Stage some changes first")
		}

		if utils.IsInteractive() {
			confirmed, err := tui.PromptConfirm("No changes staged. Would you like to stage all changes and analyze them?", true)
			if err != nil || !confirmed {
				return nil, fmt.Errorf("no staged changes to analyze")
			}
			if err := git.StageAll(ctx.Context); err != nil {
				return nil, fmt.Errorf("failed to stage changes: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no staged changes to analyze")
		}
	}

	// Get staged diff
	diff, err := git.GetStagedDiff(ctx.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	if diff == "" {
		return nil, fmt.Errorf("no staged changes found")
	}

	splog.Info("🤖 Analyzing staged changes...")
	suggestion, err := opts.AIClient.GenerateStackSuggestion(ctx.Context, diff)
	if err != nil {
		return nil, fmt.Errorf("failed to generate stack suggestion: %w", err)
	}

	return suggestion, nil
}

// CreateStackAction creates a stack of branches from a suggestion
func CreateStackAction(ctx *runtime.Context, suggestion *ai.StackSuggestion, verbose int) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch as starting point
	startBranch, err := utils.ValidateOnBranch(ctx)
	if err != nil {
		return err
	}

	// We need to unstage everything first, because we'll stage files layer by layer
	if _, err := git.RunGitCommandRaw("reset", "HEAD", "--"); err != nil {
		return fmt.Errorf("failed to unstage changes: %w", err)
	}

	// Check if any suggested branch already exists
	allBranches := eng.AllBranchNames()
	for _, layer := range suggestion.Layers {
		for _, existing := range allBranches {
			if layer.BranchName == existing {
				return fmt.Errorf("branch %s already exists. Aborting stack creation", layer.BranchName)
			}
		}
	}

	currentParent := startBranch
	success := false
	defer func() {
		if !success {
			splog.Info("Something went wrong. You might want to manually clean up or reset to %s", startBranch)
		}
	}()

	for i, layer := range suggestion.Layers {
		splog.Info("Creating layer %d: %s", i+1, layer.BranchName)

		// 1. Stage files for this layer
		for _, file := range layer.Files {
			// Check if file exists first
			if _, err := git.RunGitCommandRaw("ls-files", "--error-unmatch", file); err != nil {
				splog.Info("Warning: file %s not found in repository. Skipping.", file)
				continue
			}

			if _, err := git.RunGitCommandRaw("add", file); err != nil {
				// Non-fatal: file might have been renamed or deleted in a way that AI didn't catch
				splog.Info("Warning: failed to stage file %s: %v", file, err)
			}
		}

		// Check if we actually staged anything
		hasStaged, err := git.HasStagedChanges(ctx.Context)
		if err != nil {
			return fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStaged {
			splog.Info("No changes staged for layer %s. Creating empty commit.", layer.BranchName)
		}

		// 2. Create and checkout new branch
		if err := git.CreateAndCheckoutBranch(ctx.Context, layer.BranchName); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", layer.BranchName, err)
		}

		// 3. Commit
		commitMsg := layer.CommitMessage
		if commitMsg == "" {
			commitMsg = fmt.Sprintf("feat: %s", layer.BranchName)
		}

		if hasStaged {
			if err := git.Commit(commitMsg, verbose); err != nil {
				return fmt.Errorf("failed to commit to branch %s: %w", layer.BranchName, err)
			}
		} else {
			// Commit empty if nothing staged
			if _, err := git.RunGitCommandRaw("commit", "--allow-empty", "-m", commitMsg); err != nil {
				return fmt.Errorf("failed to create empty commit on branch %s: %w", layer.BranchName, err)
			}
		}

		// 4. Track branch
		if err := eng.TrackBranch(ctx.Context, layer.BranchName, currentParent); err != nil {
			splog.Info("Warning: failed to track branch %s: %v", layer.BranchName, err)
		}

		currentParent = layer.BranchName
	}

	success = true
	splog.Info("✅ Stack created successfully!")
	return nil
}
