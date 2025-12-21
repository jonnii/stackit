// Package ai provides AI-powered features for stackit, including context collection
// for PR description generation.
package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// PRContext contains all context information needed for AI-powered PR description generation
type PRContext struct {
	BranchName         string
	ParentBranchName   string
	TrunkBranchName    string
	CommitMessages     []string
	CodeDiff           string
	ChangedFiles       []string
	ParentPRInfo       *engine.PrInfo
	ChildPRInfo        *engine.PrInfo
	RelatedPRs         []RelatedPR
	ProjectConventions string
}

// RelatedPR represents a related PR in the stack
type RelatedPR struct {
	BranchName string
	Title      string
	URL        string
	Number     int
}

// CollectPRContext collects all necessary information for AI-powered PR description generation
func CollectPRContext(ctx *runtime.Context, eng engine.Engine, branchName string) (*PRContext, error) {
	prCtx := &PRContext{
		BranchName: branchName,
	}

	// Get branch relationships
	prCtx.TrunkBranchName = eng.Trunk()
	parentBranch := eng.GetParent(branchName)
	prCtx.ParentBranchName = parentBranch

	// Collect commit messages
	commitMessages, err := git.GetCommitMessages(ctx.Context, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	prCtx.CommitMessages = commitMessages

	// Get code diff
	codeDiff, err := getCodeDiff(ctx.Context, eng, branchName, parentBranch)
	if err != nil {
		// Non-fatal: continue without diff
		prCtx.CodeDiff = ""
	} else {
		prCtx.CodeDiff = codeDiff
	}

	// Get changed files
	changedFiles, err := getChangedFiles(ctx.Context, eng, branchName, parentBranch)
	if err != nil {
		// Non-fatal: continue without changed files
		prCtx.ChangedFiles = []string{}
	} else {
		prCtx.ChangedFiles = changedFiles
	}

	// Get parent PR info
	if parentBranch != "" {
		parentPRInfo, err := eng.GetPrInfo(ctx.Context, parentBranch)
		if err == nil && parentPRInfo != nil {
			prCtx.ParentPRInfo = parentPRInfo
		}
	}

	// Get child PR info (first child if exists)
	children := eng.GetChildren(branchName)
	if len(children) > 0 {
		childPRInfo, err := eng.GetPrInfo(ctx.Context, children[0])
		if err == nil && childPRInfo != nil {
			prCtx.ChildPRInfo = childPRInfo
		}
	}

	// Collect related PRs in stack
	prCtx.RelatedPRs = collectRelatedPRs(ctx.Context, eng, branchName)

	// Read project conventions
	repoRoot := ctx.RepoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = git.GetRepoRoot()
		if err != nil {
			// Non-fatal: continue without conventions
			prCtx.ProjectConventions = ""
		}
	}
	if repoRoot != "" {
		conventions, err := readProjectConventions(repoRoot)
		if err == nil {
			prCtx.ProjectConventions = conventions
		}
	}

	return prCtx, nil
}

// getCodeDiff returns the diff between parent and branch
func getCodeDiff(ctx context.Context, eng engine.Engine, branchName, parentBranch string) (string, error) {
	branchRevision, err := eng.GetRevision(ctx, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get branch revision: %w", err)
	}

	var baseRevision string
	if eng.IsTrunk(branchName) {
		// For trunk, use parent commit
		parentSHA, err := git.GetCommitSHA(branchName, 1)
		if err != nil {
			return "", fmt.Errorf("failed to get parent commit: %w", err)
		}
		baseRevision = parentSHA
	} else if parentBranch == "" {
		// Branch has no parent (shouldn't happen for tracked branches, but handle gracefully)
		return "", fmt.Errorf("branch has no parent and is not trunk")
	} else {
		// For regular branches, get parent revision from metadata
		meta, err := git.ReadMetadataRef(branchName)
		if err != nil || meta.ParentBranchRevision == nil {
			return "", fmt.Errorf("failed to get parent revision: %w", err)
		}
		baseRevision = *meta.ParentBranchRevision
	}

	// Get diff without color codes for AI processing
	// Use raw command to avoid color codes
	diffOutput, err := git.RunGitCommandRawWithContext(ctx, "diff", "--no-ext-diff", baseRevision, branchRevision, "--")
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return diffOutput, nil
}

// getChangedFiles returns the list of files changed between parent and branch
func getChangedFiles(ctx context.Context, eng engine.Engine, branchName, parentBranch string) ([]string, error) {
	branchRevision, err := eng.GetRevision(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch revision: %w", err)
	}

	var baseRevision string
	if eng.IsTrunk(branchName) {
		// For trunk, use parent commit
		parentSHA, err := git.GetCommitSHA(branchName, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent commit: %w", err)
		}
		baseRevision = parentSHA
	} else if parentBranch == "" {
		// Branch has no parent (shouldn't happen for tracked branches, but handle gracefully)
		return nil, fmt.Errorf("branch has no parent and is not trunk")
	} else {
		// For regular branches, get parent revision from metadata
		meta, err := git.ReadMetadataRef(branchName)
		if err != nil || meta.ParentBranchRevision == nil {
			return nil, fmt.Errorf("failed to get parent revision: %w", err)
		}
		baseRevision = *meta.ParentBranchRevision
	}

	changedFiles, err := git.GetChangedFiles(ctx, baseRevision, branchRevision)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	return changedFiles, nil
}

// collectRelatedPRs traverses the stack to find related PRs
func collectRelatedPRs(ctx context.Context, eng engine.Engine, branchName string) []RelatedPR {
	// Get full stack (parents and children)
	scope := engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    false, // Don't include current branch
		RecursiveChildren: true,
	}
	relatedBranches := eng.GetRelativeStack(branchName, scope)

	relatedPRs := make([]RelatedPR, 0, len(relatedBranches))
	for _, relatedBranch := range relatedBranches {
		prInfo, err := eng.GetPrInfo(ctx, relatedBranch)
		if err != nil || prInfo == nil {
			continue
		}

		// Only include PRs that are open or merged (not closed)
		if prInfo.State != "OPEN" && prInfo.State != "MERGED" {
			continue
		}

		relatedPR := RelatedPR{
			BranchName: relatedBranch,
			Title:      prInfo.Title,
			URL:        prInfo.URL,
		}
		if prInfo.Number != nil {
			relatedPR.Number = *prInfo.Number
		}

		relatedPRs = append(relatedPRs, relatedPR)
	}

	return relatedPRs
}

// readProjectConventions reads CONTRIBUTING.md from repo root
func readProjectConventions(repoRoot string) (string, error) {
	var conventions []string

	// Read CONTRIBUTING.md
	contributingPath := filepath.Join(repoRoot, "CONTRIBUTING.md")
	if content, err := os.ReadFile(contributingPath); err == nil {
		conventions = append(conventions, "=== CONTRIBUTING.md ===\n"+string(content))
	}

	if len(conventions) == 0 {
		return "", fmt.Errorf("no convention files found")
	}

	return strings.Join(conventions, "\n\n"), nil
}
