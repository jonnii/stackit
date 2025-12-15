package utils

import (
	"regexp"
	"strings"
)

const (
	// MaxBranchNameByteLength is the maximum length for a branch name
	// Git refs have a max length of 256 bytes, minus 22 for "refs/branch-metadata/"
	MaxBranchNameByteLength = 234
)

var (
	// BranchNameReplaceRegex matches characters that are not valid in branch names
	// Valid characters: letters, numbers, -, _, /, .
	BranchNameReplaceRegex = regexp.MustCompile(`[^-_/.a-zA-Z0-9]+`)
	
	// BranchNameIgnoreRegex matches trailing slashes and dots that should be removed
	BranchNameIgnoreRegex = regexp.MustCompile(`[/.]*$`)
)

// SanitizeBranchName sanitizes a branch name by replacing invalid characters
func SanitizeBranchName(name string) string {
	// Remove trailing slashes and dots
	name = BranchNameIgnoreRegex.ReplaceAllString(name, "")
	
	// Replace invalid characters with hyphens
	name = BranchNameReplaceRegex.ReplaceAllString(name, "-")
	
	// Remove multiple consecutive hyphens
	hyphenRegex := regexp.MustCompile(`-+`)
	name = hyphenRegex.ReplaceAllString(name, "-")
	
	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")
	
	// Limit length
	if len(name) > MaxBranchNameByteLength {
		name = name[:MaxBranchNameByteLength]
		// Trim trailing hyphen if we cut at a hyphen
		name = strings.TrimSuffix(name, "-")
	}
	
	return name
}

// GenerateBranchNameFromMessage generates a branch name from a commit message
func GenerateBranchNameFromMessage(message string) string {
	if message == "" {
		return ""
	}
	
	// Take first line of message (subject line)
	lines := strings.Split(message, "\n")
	subject := strings.TrimSpace(lines[0])
	
	// Remove common prefixes like "feat:", "fix:", etc. if present
	subject = regexp.MustCompile(`^(feat|fix|chore|docs|style|refactor|perf|test|build|ci):\s*`).ReplaceAllString(subject, "")
	
	// Sanitize and return
	return SanitizeBranchName(subject)
}

