package git

import (
	"fmt"
	"regexp"
	"strings"
)

// Hunk represents a single hunk of changes in a diff
type Hunk struct {
	File     string // File path
	OldStart int    // Line number in old file (1-indexed)
	OldCount int    // Number of lines in old file
	NewStart int    // Line number in new file (1-indexed)
	NewCount int    // Number of lines in new file
	Content  string // The actual diff content (including header)
}

// ParseStagedHunks parses the output of `git diff --cached` into structured hunks
func ParseStagedHunks() ([]Hunk, error) {
	diffOutput, err := GetStagedDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	if strings.TrimSpace(diffOutput) == "" {
		return []Hunk{}, nil
	}

	var hunks []Hunk
	lines := strings.Split(diffOutput, "\n")

	// Regex to match hunk headers: @@ -old_start,old_count +new_start,new_count @@
	// Example: @@ -10,5 +10,6 @@
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	var currentHunk *Hunk
	var currentFile string
	var hunkLines []string

	for _, line := range lines {
		// Check for file header (starts with "diff --git" or "--- a/" or "+++ b/")
		if strings.HasPrefix(line, "diff --git") {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
				hunkLines = nil
			}
			// Extract file path from "diff --git a/path b/path"
			// Format: "diff --git a/path/to/file b/path/to/file"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// parts[2] = "a/path/to/file", parts[3] = "b/path/to/file"
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
				hunks = append(hunks, *currentHunk)
			}

			// Parse hunk header
			oldStart := parseInt(match[1])
			oldCount := parseInt(match[2])
			if oldCount == 0 {
				oldCount = 1 // Default to 1 if not specified
			}
			newStart := parseInt(match[3])
			newCount := parseInt(match[4])
			if newCount == 0 {
				newCount = 1 // Default to 1 if not specified
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
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

// parseInt parses a string to int, returns 0 if empty or invalid
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
