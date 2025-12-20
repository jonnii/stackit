// Package utils provides common utility functions for the stackit project.
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

	// Remove common prefixes like "feat:", "fix:", etc. if present (with optional scope)
	subject = regexp.MustCompile(`^(feat|fix|chore|docs|style|refactor|perf|test|build|ci)(\([^)]+\))?:\s*`).ReplaceAllString(subject, "")

	// Truncate to a reasonable length for branch names (before sanitization)
	// Aim for ~50 characters to leave room for username/date prefixes
	maxSubjectLength := 50
	if len(subject) > maxSubjectLength {
		// Try to truncate at word boundary
		truncated := subject[:maxSubjectLength]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > maxSubjectLength/2 {
			// If we can find a space in the second half, truncate there
			subject = truncated[:lastSpace]
		} else {
			// Otherwise just truncate
			subject = truncated
		}
	}

	// Sanitize and return
	return SanitizeBranchName(subject)
}

// ProcessBranchNamePattern processes a branch name pattern by replacing placeholders
// Supported placeholders:
//   - {username}: The sanitized Git username
//   - {date}: Current date and time in yyyyMMddHHmmss format in UTC
//   - {message}: The sanitized commit message subject (required)
//
// The pattern must contain {message} placeholder. The pattern is processed and then
// sanitized to ensure it's a valid branch name.
func ProcessBranchNamePattern(pattern string, username, date, message string) string {
	if pattern == "" {
		// If pattern is empty, just use the message (backward compatibility)
		return GenerateBranchNameFromMessage(message)
	}

	// Validate that pattern contains {message} placeholder
	if !strings.Contains(pattern, "{message}") {
		// Fallback to just the message if pattern doesn't contain {message}
		// This should not happen if validation in SetBranchNamePattern works correctly
		return GenerateBranchNameFromMessage(message)
	}

	// Extract message subject for {message} placeholder
	messageSubject := GenerateBranchNameFromMessage(message)

	// Replace placeholders
	result := pattern
	result = strings.ReplaceAll(result, "{username}", SanitizeBranchName(username))
	result = strings.ReplaceAll(result, "{date}", date)
	result = strings.ReplaceAll(result, "{message}", messageSubject)

	// Sanitize the final result
	return SanitizeBranchName(result)
}
