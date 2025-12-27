package git

import (
	"fmt"
	"regexp"
	"strings"
)

// HunkTarget represents a hunk and its target commit
type HunkTarget struct {
	Hunk        Hunk
	CommitSHA   string
	CommitIndex int // Index in the commit list (0 = newest)
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

// hunkOverlaps checks if two hunks have overlapping line ranges.
// It includes a safety margin to account for git context lines.
func hunkOverlaps(h1, h2 Hunk) bool {
	if h1.File != h2.File {
		return false
	}

	// Staged hunk (h1) is being passed back over commit hunk (h2).
	// h1.Old is the state after h2 was applied (h2.New).
	// We add a safety margin of 3 lines (typical git context) to avoid conflicts.
	margin := 3

	h1Start := h1.OldStart - margin
	h1End := h1.OldStart + h1.OldCount + margin
	h2Start := h2.NewStart
	h2End := h2.NewStart + h2.NewCount

	overlap := h1Start <= h2End && h2Start <= h1End
	return overlap
}

// GetCommitDiff returns the diff for a commit
func GetCommitDiff(commitSHA, parentSHA string) (string, error) {
	return RunGitCommand("diff", parentSHA, commitSHA)
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

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

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
