package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

// HunkTarget represents a hunk and its target commit
type HunkTarget struct {
	Hunk        Hunk
	CommitSHA   string
	CommitIndex int // Index in the commit list (0 = newest)
}

// FindTargetCommitForHunk finds the first commit downstack where the hunk doesn't commute
// Returns the commit SHA and index, or empty string if hunk commutes with all commits
func FindTargetCommitForHunk(hunk Hunk, commitSHAs []string) (string, int, error) {
	if len(commitSHAs) == 0 {
		return "", -1, nil
	}

	// Iterate through commits from newest to oldest
	for i, commitSHA := range commitSHAs {
		// Get parent commit SHA
		parentSHA, err := GetParentCommitSHA(commitSHA)
		if err != nil {
			// If we can't get parent, skip this commit
			continue
		}

		// Check if hunk commutes with this commit
		commutes, err := CheckCommutation(hunk, commitSHA, parentSHA)
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

// CheckCommutation checks if a hunk commutes with a commit
// Two patches commute if they don't touch overlapping lines in the same file
func CheckCommutation(hunk Hunk, commitSHA, parentSHA string) (bool, error) {
	// Get the commit's diff to see what lines it touches
	commitDiff, err := GetCommitDiff(commitSHA, parentSHA)
	if err != nil {
		return false, fmt.Errorf("failed to get commit diff: %w", err)
	}

	// If commit diff is empty, they commute
	if strings.TrimSpace(commitDiff) == "" {
		return true, nil
	}

	// Parse commit diff to get line ranges for the same file
	commitHunks := parseDiffHunks(commitDiff, hunk.File)

	// If the file doesn't appear in the commit's diff at all, they commute
	// Check if the file exists in the commit diff by searching for it
	fileInDiff := false
	for _, line := range strings.Split(commitDiff, "\n") {
		if strings.Contains(line, hunk.File) {
			fileInDiff = true
			break
		}
	}
	if !fileInDiff {
		return true, nil // File doesn't exist in commit, they commute
	}

	// If the file doesn't appear in the parsed hunks (but appears in diff),
	// it might be a rename or the parsing failed - be conservative and say they don't commute
	if len(commitHunks) == 0 {
		return false, nil // File in diff but no hunks - be conservative
	}

	// Check if any commit hunk overlaps with the staged hunk in the same file
	for _, commitHunk := range commitHunks {
		if commitHunk.File != hunk.File {
			continue
		}

		// Check if line ranges overlap
		if hunkOverlaps(hunk, commitHunk) {
			return false, nil // They don't commute
		}
	}

	return true, nil // They commute
}

// hunkOverlaps checks if two hunks have overlapping line ranges
func hunkOverlaps(h1, h2 Hunk) bool {
	if h1.File != h2.File {
		return false
	}

	// Check if old line ranges overlap
	h1OldEnd := h1.OldStart + h1.OldCount - 1
	h2OldEnd := h2.OldStart + h2.OldCount - 1
	oldOverlap := !(h1OldEnd < h2.OldStart || h2OldEnd < h1.OldStart)

	// Check if new line ranges overlap
	h1NewEnd := h1.NewStart + h1.NewCount - 1
	h2NewEnd := h2.NewStart + h2.NewCount - 1
	newOverlap := !(h1NewEnd < h2.NewStart || h2NewEnd < h1.NewStart)

	return oldOverlap || newOverlap
}

// GetCommitDiff returns the diff for a commit
func GetCommitDiff(commitSHA, parentSHA string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	h1, err := resolveRefHash(repo, parentSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve parent: %w", err)
	}

	h2, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve commit: %w", err)
	}

	c1, err := repo.CommitObject(h1)
	if err != nil {
		return "", fmt.Errorf("failed to get parent commit: %w", err)
	}

	c2, err := repo.CommitObject(h2)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	patch, err := c1.Patch(c2)
	if err != nil {
		return "", fmt.Errorf("failed to get patch: %w", err)
	}

	return patch.String(), nil
}

// GetParentCommitSHA returns the parent commit SHA of a commit
func GetParentCommitSHA(commitSHA string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve commit: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	if commit.NumParents() == 0 {
		return "", fmt.Errorf("commit has no parents")
	}

	return commit.ParentHashes[0].String(), nil
}

// parseDiffHunks parses a diff output and extracts hunks for a specific file
func parseDiffHunks(diffOutput, targetFile string) []Hunk {
	if strings.TrimSpace(diffOutput) == "" {
		return []Hunk{}
	}

	var hunks []Hunk
	lines := strings.Split(diffOutput, "\n")

	// Regex to match hunk headers
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	var currentHunk *Hunk
	var currentFile string
	var hunkLines []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		// Check for file header
		if strings.HasPrefix(line, "diff --git") {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				if currentHunk.File == targetFile {
					hunks = append(hunks, *currentHunk)
				}
				currentHunk = nil
				hunkLines = nil
			}
			// Extract file path from "diff --git a/path b/path"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				bPath := parts[len(parts)-1]
				if strings.HasPrefix(bPath, "b/") {
					currentFile = strings.TrimPrefix(bPath, "b/")
				}
			}
			continue
		}

		// Check for hunk header
		if match := hunkHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				if currentHunk.File == targetFile {
					hunks = append(hunks, *currentHunk)
				}
			}

			// Parse hunk header
			oldStart := parseInt(match[1])
			oldCount := parseInt(match[2])
			if oldCount == 0 {
				oldCount = 1
			}
			newStart := parseInt(match[3])
			newCount := parseInt(match[4])
			if newCount == 0 {
				newCount = 1
			}

			currentHunk = &Hunk{
				File:     currentFile,
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
			}
			hunkLines = []string{line}
			continue
		}

		// Accumulate hunk content
		if currentHunk != nil {
			hunkLines = append(hunkLines, line)
		}
	}

	// Save last hunk
	if currentHunk != nil {
		currentHunk.Content = strings.Join(hunkLines, "\n")
		if currentHunk.File == targetFile {
			hunks = append(hunks, *currentHunk)
		}
	}

	return hunks
}

// ApplyHunksToCommit applies multiple hunks to a commit by rewriting it
func ApplyHunksToCommit(ctx context.Context, hunks []Hunk, commitSHA string, branchName string) error {
	if len(hunks) == 0 {
		return nil
	}

	// Save current branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	// Get the repo root for running git commands in the correct directory
	repoRoot, err := GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repo root: %w", err)
	}

	// Get commit's parent
	parentSHA, err := GetParentCommitSHA(commitSHA)
	if err != nil {
		return fmt.Errorf("failed to get parent commit: %w", err)
	}

	// Get commit message
	commitMessage, err := GetCommitMessage(commitSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit message: %w", err)
	}

	// Get the commit author and date
	author, err := GetCommitAuthorFromSHA(commitSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit author: %w", err)
	}

	date, err := GetCommitDateFromSHA(commitSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit date: %w", err)
	}

	// Create a temporary directory for patch files (unique per operation to avoid conflicts)
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("stackit-absorb-%s-*", commitSHA[:8]))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	patchFile := filepath.Join(tmpDir, "hunks.patch")

	// Group hunks by file
	hunksByFile := make(map[string][]Hunk)
	for _, hunk := range hunks {
		hunksByFile[hunk.File] = append(hunksByFile[hunk.File], hunk)
	}

	// Construct patch manually from hunks
	// This is more precise than GetStagedDiff() because it only includes
	// the hunks that were matched to THIS commit.
	var patchContent strings.Builder
	for file, fileHunks := range hunksByFile {
		// We need to provide enough headers for git apply --cached to work.
		// For a simple hunk application, we need the diff --git, ---, and +++ headers.
		patchContent.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
		patchContent.WriteString(fmt.Sprintf("--- a/%s\n", file))
		patchContent.WriteString(fmt.Sprintf("+++ b/%s\n", file))
		for _, hunk := range fileHunks {
			// hunk.Content already includes the @@ header and lines
			patchContent.WriteString(hunk.Content)
			if !strings.HasSuffix(hunk.Content, "\n") {
				patchContent.WriteString("\n")
			}
		}
	}
	if err := os.WriteFile(patchFile, []byte(patchContent.String()), 0600); err != nil {
		return fmt.Errorf("failed to write patch file: %w", err)
	}

	// Checkout parent commit (detached HEAD)
	if err := CheckoutDetached(ctx, parentSHA); err != nil {
		// Restore branch
		if currentBranch != "" {
			_ = CheckoutBranch(ctx, currentBranch)
		}
		return fmt.Errorf("failed to checkout parent: %w", err)
	}

	// Cleanup logic to ensure we always try to get back to original branch
	defer func() {
		// Use git command directly to avoid go-git caching issues
		nowBranch, _ := RunGitCommandWithContext(ctx, "branch", "--show-current")
		nowBranch = strings.TrimSpace(nowBranch)

		if nowBranch != currentBranch && currentBranch != "" {
			// Clean up index/working tree if needed before checkout
			_, _ = RunGitCommandWithContext(ctx, "reset", "--hard", "HEAD")
			_ = CheckoutBranch(ctx, currentBranch)
		}
	}()

	// First, apply the original commit's changes to the index
	commitDiff, err := GetCommitDiff(commitSHA, parentSHA)
	if err != nil {
		return fmt.Errorf("failed to get commit diff: %w", err)
	}

	// Create temporary file for commit diff
	commitPatchFile := filepath.Join(tmpDir, "commit.patch")
	if err := os.WriteFile(commitPatchFile, []byte(commitDiff), 0600); err != nil {
		return fmt.Errorf("failed to write commit patch file: %w", err)
	}

	// Apply the original commit's changes to the index
	cmd := exec.Command("git", "apply", "--cached", commitPatchFile)
	cmd.Dir = repoRoot
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply commit diff: %w (stderr: %s)", err, stderr.String())
	}

	// Now apply the hunks patch to the index
	cmd = exec.Command("git", "apply", "--cached", patchFile)
	cmd.Dir = repoRoot
	stderr.Reset()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply patch: %w (stderr: %s)", err, stderr.String())
	}

	// Create new commit with same message, author, and date
	env := os.Environ()
	env = append(env, fmt.Sprintf("GIT_AUTHOR_NAME=%s", author.Name))
	env = append(env, fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", author.Email))
	env = append(env, fmt.Sprintf("GIT_AUTHOR_DATE=%s", date.Format("2006-01-02T15:04:05-0700")))
	env = append(env, fmt.Sprintf("GIT_COMMITTER_NAME=%s", author.Name))
	env = append(env, fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", author.Email))
	env = append(env, fmt.Sprintf("GIT_COMMITTER_DATE=%s", date.Format("2006-01-02T15:04:05-0700")))

	cmd = exec.Command("git", "commit", "-m", commitMessage)
	cmd.Dir = repoRoot
	cmd.Env = env
	stderr.Reset()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w (stderr: %s)", err, stderr.String())
	}

	// Get new commit SHA
	newCommitSHA, err := RunGitCommandWithContext(ctx, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get new commit SHA: %w", err)
	}

	// Update branch to point to new commit
	if err := UpdateBranchRef(branchName, strings.TrimSpace(newCommitSHA)); err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	return nil
}

// CheckoutDetached checks out a commit in detached HEAD state
func CheckoutDetached(ctx context.Context, commitSHA string) error {
	_, err := RunGitCommandWithContext(ctx, "checkout", commitSHA)
	if err != nil {
		return fmt.Errorf("failed to checkout commit: %w", err)
	}
	return nil
}

// GetCommitMessage returns the full commit message for a commit
func GetCommitMessage(commitSHA string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve commit: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return strings.TrimSpace(commit.Message), nil
}

// CommitAuthor represents a commit author
type CommitAuthor struct {
	Name  string
	Email string
}

// GetCommitAuthorFromSHA returns the author for a commit
func GetCommitAuthorFromSHA(commitSHA string) (*CommitAuthor, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commit: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &CommitAuthor{
		Name:  commit.Author.Name,
		Email: commit.Author.Email,
	}, nil
}

// GetCommitDateFromSHA returns the commit date
func GetCommitDateFromSHA(commitSHA string) (time.Time, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return time.Time{}, err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to resolve commit: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.When, nil
}

// UpdateBranchRef updates a branch reference to point to a new commit
func UpdateBranchRef(branchName, commitSHA string) error {
	repo, err := GetDefaultRepo()
	if err != nil {
		return err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return fmt.Errorf("failed to resolve commit SHA: %w", err)
	}

	// Update the reference
	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	ref := plumbing.NewHashReference(refName, hash)
	if err := repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	return nil
}
