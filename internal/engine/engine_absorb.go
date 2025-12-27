package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stackit.dev/stackit/internal/git"
)

// ApplyHunksToBranch applies multiple hunks to commits in a branch by recreating them.
func (e *engineImpl) ApplyHunksToBranch(ctx context.Context, branch Branch, hunksByCommit map[string][]git.Hunk) error {
	if len(hunksByCommit) == 0 {
		return nil
	}

	branchName := branch.GetName()

	// Save current state to restore later
	originalRev, _ := e.git.RunGitCommandWithContext(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if originalRev == "HEAD" || originalRev == "" {
		originalRev, _ = e.git.RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	}
	originalRev = strings.TrimSpace(originalRev)

	defer func() {
		if originalRev != "" {
			_, _ = e.git.RunGitCommandWithContext(ctx, "checkout", originalRev)
		}
	}()

	// Get all commits in the branch
	commitSHAs, err := branch.GetAllCommits(CommitFormatSHA)
	if err != nil {
		return fmt.Errorf("failed to get commits for branch %s: %w", branchName, err)
	}

	if len(commitSHAs) == 0 {
		return fmt.Errorf("branch %s has no commits", branchName)
	}

	// Get base revision (parent of the first commit in the branch)
	meta, err := e.readMetadataRef(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata for %s: %w", branchName, err)
	}
	if meta.ParentBranchRevision == nil {
		return fmt.Errorf("branch %s has no parent revision", branchName)
	}
	currentBase := *meta.ParentBranchRevision

	// Checkout base in detached HEAD
	if _, err := e.git.RunGitCommandWithContext(ctx, "checkout", "--detach", currentBase); err != nil {
		return fmt.Errorf("failed to checkout base %s: %w", currentBase[:8], err)
	}

	// Recreate branch commit by commit (oldest to newest)
	for i := len(commitSHAs) - 1; i >= 0; i-- {
		commitSHA := commitSHAs[i]
		hunks, hasHunks := hunksByCommit[commitSHA]

		// 1. Cherry-pick the original commit
		if _, err := e.git.RunGitCommandWithContext(ctx, "cherry-pick", commitSHA); err != nil {
			_, _ = e.git.RunGitCommandWithContext(ctx, "cherry-pick", "--abort")
			return fmt.Errorf("failed to cherry-pick %s: %w", commitSHA[:8], err)
		}

		if hasHunks {
			// 2. Apply hunks to this commit
			tmpDir, err := os.MkdirTemp("", fmt.Sprintf("stackit-absorb-%s-*", commitSHA[:8]))
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			patchFile := filepath.Join(tmpDir, "hunks.patch")
			var patchContent strings.Builder
			hunksByFile := make(map[string][]git.Hunk)
			for _, hunk := range hunks {
				hunksByFile[hunk.File] = append(hunksByFile[hunk.File], hunk)
			}
			for file, fileHunks := range hunksByFile {
				patchContent.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
				patchContent.WriteString(fmt.Sprintf("--- a/%s\n", file))
				patchContent.WriteString(fmt.Sprintf("+++ b/%s\n", file))
				for _, hunk := range fileHunks {
					patchContent.WriteString(hunk.Content)
					if !strings.HasSuffix(hunk.Content, "\n") {
						patchContent.WriteString("\n")
					}
				}
			}
			if err := os.WriteFile(patchFile, []byte(patchContent.String()), 0600); err != nil {
				return fmt.Errorf("failed to write hunks patch: %w", err)
			}

			// Apply hunks to the worktree and index
			if _, err := e.git.RunGitCommandWithContext(ctx, "apply", patchFile); err != nil {
				return fmt.Errorf("failed to apply hunks for commit %s: %w", commitSHA[:8], err)
			}

			// 3. Amend the commit
			if _, err := e.git.RunGitCommandWithContext(ctx, "commit", "-a", "--amend", "--no-edit", "--no-verify"); err != nil {
				return fmt.Errorf("failed to amend commit %s: %w", commitSHA[:8], err)
			}
		}
	}

	// Get new tip
	newTip, err := e.git.RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get new tip: %w", err)
	}
	newTip = strings.TrimSpace(newTip)

	// Update branch to point to new tip
	if err := e.git.UpdateBranchRef(branchName, newTip); err != nil {
		return fmt.Errorf("failed to update branch %s: %w", branchName, err)
	}

	return nil
}

// FindTargetCommitForHunk finds the first commit downstack where the hunk doesn't commute
func (e *engineImpl) FindTargetCommitForHunk(hunk git.Hunk, commitSHAs []string) (string, int, error) {
	if len(commitSHAs) == 0 {
		return "", -1, nil
	}

	// Iterate through commits from newest to oldest
	for i, commitSHA := range commitSHAs {
		// Get parent commit SHA
		parentSHA, err := e.git.GetParentCommitSHA(commitSHA)
		if err != nil {
			// If we can't get parent, skip this commit
			continue
		}

		// Check if hunk commutes with this commit
		commutes, err := e.git.CheckCommutation(hunk, commitSHA, parentSHA)
		if err != nil {
			return "", -1, fmt.Errorf("failed to check commutation: %w", err)
		}

		if !commutes {
			// Found the target commit - hunk doesn't commute with it
			return commitSHA, i, nil
		}
	}

	// Hunk commutes with all commits
	return "", -1, nil
}
