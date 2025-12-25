package merge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

// ConsolidationResult contains information about a completed consolidation
type ConsolidationResult struct {
	BranchName string
	PRNumber   int
	PRURL      string
}

// ConsolidateMergeExecutor handles stack consolidation merging
type ConsolidateMergeExecutor struct {
	plan         *Plan
	githubClient github.Client
	engine       mergeExecuteEngine
	splog        *tui.Splog
	repoRoot     string
}

// NewConsolidateMergeExecutor creates a new consolidation executor
func NewConsolidateMergeExecutor(plan *Plan, githubClient github.Client, engine mergeExecuteEngine, splog *tui.Splog, repoRoot string) *ConsolidateMergeExecutor {
	return &ConsolidateMergeExecutor{
		plan:         plan,
		githubClient: githubClient,
		engine:       engine,
		splog:        splog,
		repoRoot:     repoRoot,
	}
}

// Execute performs stack consolidation merging
func (c *ConsolidateMergeExecutor) Execute(ctx context.Context, opts ExecuteOptions) (*ConsolidationResult, error) {
	c.splog.Info("🔀 Starting stack consolidation merge...")

	// Phase 1: Pre-validate the stack
	if err := c.preValidateStack(ctx, opts.Force); err != nil {
		return nil, fmt.Errorf("pre-validation failed: %w", err)
	}

	// Show what will be consolidated
	c.splog.Info("Stack to consolidate:")
	for i, branchInfo := range c.plan.BranchesToMerge {
		symbol := "  ○"
		if i == len(c.plan.BranchesToMerge)-1 {
			symbol = "  ◉"
		}
		c.splog.Info("%s %s PR #%d", symbol, branchInfo.BranchName, branchInfo.PRNumber)
	}
	c.splog.Info("    ↓")
	c.splog.Info("  📦 Consolidated PR")
	c.splog.Newline()

	// Phase 2: Create consolidation branch
	consolidationBranch, err := c.createConsolidationBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create consolidation branch: %w", err)
	}

	// Phase 3: Create consolidation PR
	pr, err := c.createConsolidationPR(ctx, consolidationBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to create consolidation PR: %w", err)
	}

	c.splog.Info("✅ Created consolidation branch: %s", consolidationBranch)
	// Note: PR link display moved to caller to avoid TUI race conditions

	// Phase 4: Wait for CI and auto-merge
	if err := c.waitForConsolidationMerge(ctx, consolidationBranch, pr); err != nil {
		return nil, fmt.Errorf("consolidation merge failed: %w", err)
	}

	// Phase 5: Post-merge cleanup and documentation
	if err := c.postMergeCleanup(ctx); err != nil {
		c.splog.Warn("Post-merge cleanup had issues: %v", err)
		// Don't fail the entire operation for cleanup issues
	}

	c.splog.Info("🎉 Stack consolidation merge completed successfully!")

	result := &ConsolidationResult{
		BranchName: consolidationBranch,
		PRNumber:   pr.Number,
		PRURL:      pr.HTMLURL,
	}
	return result, nil
}

// preValidateStack ensures all PRs are ready for consolidation
func (c *ConsolidateMergeExecutor) preValidateStack(ctx context.Context, force bool) error {
	for _, branchInfo := range c.plan.BranchesToMerge {
		// Check PR exists and is open
		prInfo, err := c.engine.GetPrInfo(branchInfo.BranchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			return fmt.Errorf("PR not found for branch %s", branchInfo.BranchName)
		}
		if prInfo.State != prStateOpen {
			return fmt.Errorf("PR #%d for branch %s is %s (not open)", *prInfo.Number, branchInfo.BranchName, prInfo.State)
		}

		// Check if local matches remote for consolidation validation
		matchesRemote, err := c.engine.BranchMatchesRemote(branchInfo.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check remote tracking for %s: %w", branchInfo.BranchName, err)
		}
		if !matchesRemote {
			// Get detailed difference information like the planning phase does
			diffInfo := getBranchRemoteDifference(branchInfo.BranchName, c.splog)
			if diffInfo != "" {
				if !force {
					return fmt.Errorf("branch %s differs from remote: %s, use --force to proceed", branchInfo.BranchName, diffInfo)
				}
				c.splog.Warn("Branch %s differs from remote: %s, but proceeding with consolidation", branchInfo.BranchName, diffInfo)
			} else {
				if !force {
					return fmt.Errorf("branch %s differs from remote, use --force to proceed", branchInfo.BranchName)
				}
				c.splog.Warn("Branch %s differs from remote, but proceeding with consolidation", branchInfo.BranchName)
			}
		}

		c.splog.Debug("✅ Branch %s is ready for consolidation", branchInfo.BranchName)
	}

	// Ensure trunk is up to date
	pullResult, err := c.engine.PullTrunk(ctx)
	if err != nil {
		return fmt.Errorf("failed to update trunk: %w", err)
	}
	if pullResult == engine.PullConflict {
		return fmt.Errorf("trunk has conflicts with remote")
	}

	return nil
}

// createConsolidationBranch creates a branch containing all stack commits
func (c *ConsolidateMergeExecutor) createConsolidationBranch(ctx context.Context) (string, error) {
	// Generate unique branch name
	timestamp := time.Now().Unix()
	scope := c.getStackScope()
	branchName := fmt.Sprintf("stack-consolidate-%s-%d", scope, timestamp)

	c.splog.Info("📋 Creating consolidation branch: %s", branchName)

	// Start from trunk
	if err := git.CreateAndCheckoutBranch(ctx, branchName); err != nil {
		return "", fmt.Errorf("failed to create and checkout branch: %w", err)
	}

	// Reset to trunk (since CreateAndCheckoutBranch creates from current HEAD)
	if _, err := git.RunGitCommandWithContext(ctx, "reset", "--hard", c.engine.Trunk().Name); err != nil {
		return "", fmt.Errorf("failed to reset to trunk: %w", err)
	}

	// Merge all stack branches with --no-ff to preserve branch structure and enable auto-closing
	for i, branchInfo := range c.plan.BranchesToMerge {
		c.splog.Info("  Merging %s (%d/%d)...", branchInfo.BranchName, i+1, len(c.plan.BranchesToMerge))

		// Use merge --no-ff to preserve branch structure and allow GitHub to auto-close PRs
		commitMsg := fmt.Sprintf("Consolidate %s: %s", branchInfo.BranchName, c.getBranchTitle(branchInfo))
		if _, err := git.RunGitCommandWithContext(ctx, "merge", branchInfo.BranchName, "--no-ff", "-m", commitMsg); err != nil {
			return "", fmt.Errorf("failed to merge %s: %w", branchInfo.BranchName, err)
		}
	}

	// Push the consolidation branch
	if err := git.PushBranch(ctx, branchName, git.GetRemote(), false, false); err != nil {
		return "", fmt.Errorf("failed to push consolidation branch: %w", err)
	}

	c.splog.Info("✅ Consolidation branch created and pushed")
	return branchName, nil
}

// createConsolidationPR creates a PR for the consolidation branch
func (c *ConsolidateMergeExecutor) createConsolidationPR(ctx context.Context, branchName string) (*github.PullRequestInfo, error) {
	scope := c.getStackScope()
	title := fmt.Sprintf("[%s] Consolidate stack: %s", scope, c.getConsolidationTitle())

	body := c.buildConsolidationPRBody()

	owner, repo := c.getOwnerRepo()
	opts := github.CreatePROptions{
		Title: title,
		Body:  body,
		Head:  branchName,
		Base:  c.engine.Trunk().Name,
		Draft: false, // Ready for review
	}

	pr, err := c.githubClient.CreatePullRequest(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	return pr, nil
}

// waitForConsolidationCI waits for CI checks to pass on the consolidation PR
func (c *ConsolidateMergeExecutor) waitForConsolidationCI(ctx context.Context, branchName string, prNumber int) error {
	if c.githubClient == nil {
		return fmt.Errorf("GitHub client not available")
	}

	timeout := 10 * time.Minute // Default timeout for consolidation CI
	pollInterval := 15 * time.Second
	startTime := time.Now()
	deadline := startTime.Add(timeout)

	c.splog.Info("   Waiting for CI checks (timeout: %v)...", timeout)

	for {
		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for CI checks on consolidation PR #%d after %v", prNumber, timeout)
		}

		// Check CI status
		status, err := c.githubClient.GetPRChecksStatus(ctx, branchName)
		if err != nil {
			c.splog.Debug("Error checking CI status: %v", err)
			// Continue polling on transient errors
		} else {
			if !status.Passing {
				// CI checks failed
				return fmt.Errorf("CI checks failed on consolidation PR #%d", prNumber)
			}
			if !status.Pending {
				// All checks passed and none are pending
				elapsed := time.Since(startTime)
				c.splog.Info("✅ CI checks passed on consolidation PR #%d after %v", prNumber, elapsed.Round(time.Second))
				return nil
			}
			// Still pending, continue waiting
		}

		// Wait before next check
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// waitForConsolidationMerge waits for CI to pass and auto-merges the consolidation PR
func (c *ConsolidateMergeExecutor) waitForConsolidationMerge(ctx context.Context, branchName string, pr *github.PullRequestInfo) error {
	c.splog.Info("Consolidation PR:")
	c.splog.Info("  ◉ %s PR #%d ⏳", branchName, pr.Number)
	c.splog.Info("     %s", pr.HTMLURL)
	c.splog.Info("     Waiting for CI checks to pass...")

	// Wait for CI checks to pass
	if err := c.waitForConsolidationCI(ctx, branchName, pr.Number); err != nil {
		return fmt.Errorf("CI checks failed for consolidation PR: %w", err)
	}

	// Auto-merge the PR
	c.splog.Info("Consolidation PR:")
	c.splog.Info("  ◉ %s PR #%d ✓", branchName, pr.Number)
	c.splog.Info("     Auto-merging...")

	if err := c.githubClient.MergePullRequest(ctx, branchName); err != nil {
		return fmt.Errorf("failed to auto-merge consolidation PR #%d: %w", pr.Number, err)
	}

	c.splog.Info("✅ Consolidation PR #%d has been merged automatically!", pr.Number)

	// Show final state
	c.splog.Info("Consolidation complete:")
	c.splog.Info("  ✓ %s (merged)", branchName)

	return nil
}

// postMergeCleanup handles cleanup and documentation after successful merge
func (c *ConsolidateMergeExecutor) postMergeCleanup(ctx context.Context) error {
	c.splog.Info("🧹 Running post-merge cleanup...")

	// Update individual PR descriptions
	c.updateIndividualPRs(ctx)

	// Restack remaining branches
	if err := c.restackRemainingBranches(ctx); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}

// updateIndividualPRs adds documentation to individual PRs explaining the consolidation
func (c *ConsolidateMergeExecutor) updateIndividualPRs(ctx context.Context) {
	consolidationInfo := c.getConsolidationInfo()

	for _, branchInfo := range c.plan.BranchesToMerge {
		prInfo, err := c.engine.GetPrInfo(branchInfo.BranchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			continue // PR might already be gone
		}

		// Add consolidation notice to PR body
		updatedBody := c.addConsolidationNotice(prInfo.Body, consolidationInfo)

		updateOpts := github.UpdatePROptions{
			Body: &updatedBody,
		}

		owner, repo := c.getOwnerRepo()
		if err := c.githubClient.UpdatePullRequest(ctx, owner, repo, *prInfo.Number, updateOpts); err != nil {
			c.splog.Warn("Failed to update PR #%d: %v", *prInfo.Number, err)
		} else {
			c.splog.Debug("✅ Updated documentation for PR #%d", *prInfo.Number)
		}
	}
}

// restackRemainingBranches updates any remaining branches in the stack
func (c *ConsolidateMergeExecutor) restackRemainingBranches(ctx context.Context) error {
	// Pull latest trunk changes first
	if _, err := c.engine.PullTrunk(ctx); err != nil {
		return err
	}

	if len(c.plan.UpstackBranches) == 0 {
		return nil
	}

	branches := make([]engine.Branch, len(c.plan.UpstackBranches))
	for i, name := range c.plan.UpstackBranches {
		branches[i] = c.engine.GetBranch(name)
	}

	batchResult, err := c.engine.RestackBranches(ctx, branches)
	if err != nil {
		if batchResult.ConflictBranch != "" {
			c.splog.Warn("Conflict restacking %s - manual resolution needed", batchResult.ConflictBranch)
			return nil // Don't fail the whole cleanup for a conflict
		}
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}

// Helper methods

func (c *ConsolidateMergeExecutor) getStackScope() string {
	// Get scope from the first branch in the stack
	if len(c.plan.BranchesToMerge) > 0 {
		scope := c.engine.GetScopeInternal(c.plan.BranchesToMerge[0].BranchName)
		if !scope.IsEmpty() {
			return scope.String()
		}
	}
	return "stack"
}

func (c *ConsolidateMergeExecutor) getBranchTitle(branchInfo BranchMergeInfo) string {
	prInfo, _ := c.engine.GetPrInfo(branchInfo.BranchName)
	if prInfo != nil {
		return prInfo.Title
	}
	return branchInfo.BranchName
}

func (c *ConsolidateMergeExecutor) getConsolidationTitle() string {
	if len(c.plan.BranchesToMerge) == 0 {
		return "Stack consolidation"
	}

	// Use the title of the top-most PR as the main title
	topBranch := c.plan.BranchesToMerge[len(c.plan.BranchesToMerge)-1]
	return c.getBranchTitle(topBranch)
}

func (c *ConsolidateMergeExecutor) buildConsolidationPRBody() string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("## Stack Consolidation: %s\n\n", c.getStackScope()))
	body.WriteString("This PR consolidates the following stack of changes into a single merge:\n\n")

	for i, branchInfo := range c.plan.BranchesToMerge {
		prInfo, _ := c.engine.GetPrInfo(branchInfo.BranchName)
		if prInfo != nil && prInfo.Number != nil {
			body.WriteString(fmt.Sprintf("%d. **PR #%d**: %s\n", i+1, *prInfo.Number, prInfo.Title))
		} else {
			body.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, branchInfo.BranchName, c.getBranchTitle(branchInfo)))
		}
	}

	body.WriteString("\n### Benefits\n")
	body.WriteString("- ✅ Single CI run validates entire stack\n")
	body.WriteString("- ✅ Atomic merge prevents partial stack states\n")
	body.WriteString("- ✅ Faster than sequential merging\n")
	body.WriteString("- ✅ Cleaner merge history\n")

	body.WriteString("\n### After Merge\n")
	body.WriteString("Individual PRs will be automatically documented and closed.\n")

	// Add dependency tree
	body.WriteString("\n### Stack Structure\n")
	body.WriteString("```\n")
	body.WriteString(c.buildStackTree())
	body.WriteString("```\n")

	return body.String()
}

func (c *ConsolidateMergeExecutor) buildStackTree() string {
	var tree strings.Builder
	trunkName := c.engine.Trunk().Name

	tree.WriteString(trunkName + "\n")

	for _, branchInfo := range c.plan.BranchesToMerge {
		// Get depth for indentation
		branch := c.engine.GetBranch(branchInfo.BranchName)
		depth := 0
		parent := c.engine.GetParent(branch)
		for parent != nil && !parent.IsTrunk() {
			depth++
			parent = c.engine.GetParent(*parent)
		}

		indent := strings.Repeat("  ", depth+1)
		tree.WriteString(fmt.Sprintf("%s├─ %s (PR #%d)\n", indent, branchInfo.BranchName, branchInfo.PRNumber))
	}

	return tree.String()
}

func (c *ConsolidateMergeExecutor) addConsolidationNotice(existingBody, consolidationInfo string) string {
	notice := "\n\n---\n\n## Stack Consolidation\n\n"
	notice += fmt.Sprintf("This PR was part of a stack that was consolidated into a single merge.\n\n%s\n\n", consolidationInfo)
	notice += "The stack has been merged atomically to ensure consistency."

	if existingBody == "" {
		return notice
	}

	return existingBody + notice
}

func (c *ConsolidateMergeExecutor) getConsolidationInfo() string {
	return fmt.Sprintf("Consolidated on %s as part of stack merge strategy.", time.Now().Format("2006-01-02"))
}

func (c *ConsolidateMergeExecutor) getOwnerRepo() (owner, repo string) {
	return c.githubClient.GetOwnerRepo()
}
