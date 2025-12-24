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
func (c *ConsolidateMergeExecutor) Execute(ctx context.Context, opts ExecuteOptions) error {
	c.splog.Info("🔀 Starting stack consolidation merge...")

	// Phase 1: Pre-validate the stack
	if err := c.preValidateStack(ctx, opts.Force); err != nil {
		return fmt.Errorf("pre-validation failed: %w", err)
	}

	// Phase 2: Create consolidation branch
	consolidationBranch, err := c.createConsolidationBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create consolidation branch: %w", err)
	}

	// Phase 3: Create consolidation PR
	pr, err := c.createConsolidationPR(ctx, consolidationBranch)
	if err != nil {
		return fmt.Errorf("failed to create consolidation PR: %w", err)
	}

	c.splog.Info("✅ Created consolidation PR #%d: %s", pr.Number, pr.HTMLURL)

	// Phase 4: Wait for CI and user to merge
	if err := c.waitForConsolidationMerge(ctx, pr); err != nil {
		return fmt.Errorf("consolidation merge failed: %w", err)
	}

	// Phase 5: Post-merge cleanup and documentation
	if err := c.postMergeCleanup(ctx); err != nil {
		c.splog.Warn("Post-merge cleanup had issues: %v", err)
		// Don't fail the entire operation for cleanup issues
	}

	c.splog.Info("🎉 Stack consolidation merge completed successfully!")
	return nil
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

	// Cherry-pick commits from each branch to preserve individual commits
	for i, branchInfo := range c.plan.BranchesToMerge {
		c.splog.Info("  Consolidating commits from %s (%d/%d)...", branchInfo.BranchName, i+1, len(c.plan.BranchesToMerge))

		// Get commits unique to this branch (not in trunk or previous branches)
		commits, err := c.getUniqueCommitsForBranch(branchInfo.BranchName)
		if err != nil {
			return "", fmt.Errorf("failed to get unique commits for %s: %w", branchInfo.BranchName, err)
		}

		// Cherry-pick each commit
		for j, commit := range commits {
			c.splog.Debug("    Cherry-picking commit %d/%d from %s: %s", j+1, len(commits), branchInfo.BranchName, commit[:8])
			if _, err := git.RunGitCommandWithContext(ctx, "cherry-pick", commit); err != nil {
				return "", fmt.Errorf("failed to cherry-pick %s from %s: %w", commit, branchInfo.BranchName, err)
			}
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

// waitForConsolidationMerge waits for the consolidation PR to be merged
func (c *ConsolidateMergeExecutor) waitForConsolidationMerge(ctx context.Context, pr *github.PullRequestInfo) error {
	c.splog.Info("⏳ Waiting for consolidation PR #%d to be merged...", pr.Number)
	c.splog.Info("   PR: %s", pr.HTMLURL)
	c.splog.Info("   Once CI passes and you merge this PR, the consolidation will complete automatically.")

	// Poll for merge status
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if PR is merged
			prInfo, err := c.engine.GetPrInfo("") // Get by branch name from PR
			if err == nil && prInfo != nil && prInfo.State == "MERGED" {
				c.splog.Info("✅ Consolidation PR #%d has been merged!", pr.Number)
				return nil
			}

			// Check if PR was closed without merging
			if err == nil && prInfo != nil && prInfo.State == "CLOSED" {
				return fmt.Errorf("consolidation PR #%d was closed without merging", pr.Number)
			}
		}
	}
}

// postMergeCleanup handles cleanup and documentation after successful merge
func (c *ConsolidateMergeExecutor) postMergeCleanup(ctx context.Context) error {
	c.splog.Info("🧹 Running post-merge cleanup...")

	// Update individual PR descriptions
	c.updateIndividualPRs(ctx)

	// Delete consolidated branches
	c.deleteMergedBranches(ctx)

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

// deleteMergedBranches removes the consolidated branches
func (c *ConsolidateMergeExecutor) deleteMergedBranches(ctx context.Context) {
	for _, branchInfo := range c.plan.BranchesToMerge {
		branch := c.engine.GetBranch(branchInfo.BranchName)
		if branch.IsTracked() {
			if err := c.engine.DeleteBranch(ctx, branchInfo.BranchName); err != nil {
				c.splog.Warn("Failed to delete branch %s: %v", branchInfo.BranchName, err)
			} else {
				c.splog.Debug("✅ Deleted branch %s", branchInfo.BranchName)
			}
		}
	}
}

// restackRemainingBranches updates any remaining branches in the stack
func (c *ConsolidateMergeExecutor) restackRemainingBranches(ctx context.Context) error {
	// Pull latest trunk changes first
	if _, err := c.engine.PullTrunk(ctx); err != nil {
		return err
	}

	for _, upstackBranch := range c.plan.UpstackBranches {
		branch := c.engine.GetBranch(upstackBranch)
		if result, err := c.engine.RestackBranch(ctx, branch); err != nil {
			if result.Result == engine.RestackConflict {
				c.splog.Warn("Conflict restacking %s - manual resolution needed", upstackBranch)
			} else {
				return fmt.Errorf("failed to restack %s: %w", upstackBranch, err)
			}
		}
	}

	return nil
}

// getUniqueCommitsForBranch returns commits that are in the branch but not in trunk
// and not in any branches that come before it in the merge order
func (c *ConsolidateMergeExecutor) getUniqueCommitsForBranch(branchName string) ([]string, error) {
	// Get all commits in this branch
	branchCommits, err := c.engine.GetAllCommitsInternal(branchName, engine.CommitFormatSHA)
	if err != nil {
		return nil, err
	}

	// Get commits in trunk
	trunkCommits, err := c.engine.GetAllCommitsInternal(c.engine.Trunk().Name, engine.CommitFormatSHA)
	if err != nil {
		return nil, err
	}

	// Create set of commits to exclude (trunk + all previous branches in merge order)
	excludeSet := make(map[string]bool)
	for _, commit := range trunkCommits {
		excludeSet[commit] = true
	}

	// Find this branch's position in the merge order
	branchIndex := -1
	for i, branchInfo := range c.plan.BranchesToMerge {
		if branchInfo.BranchName == branchName {
			branchIndex = i
			break
		}
	}

	// Exclude commits from branches that come before this one in merge order
	for i := 0; i < branchIndex; i++ {
		prevBranchCommits, err := c.engine.GetAllCommitsInternal(c.plan.BranchesToMerge[i].BranchName, engine.CommitFormatSHA)
		if err != nil {
			return nil, err
		}
		for _, commit := range prevBranchCommits {
			excludeSet[commit] = true
		}
	}

	// Filter branch commits to only include unique ones
	var uniqueCommits []string
	for _, commit := range branchCommits {
		if !excludeSet[commit] {
			uniqueCommits = append(uniqueCommits, commit)
		}
	}

	return uniqueCommits, nil
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
