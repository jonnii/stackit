package ai

import (
	"fmt"
	"strings"
)

const prNumberNA = "N/A"

const (
	// maxDiffSize is the maximum size of diff to include in prompt (in characters)
	// If diff exceeds this, we'll include a summary instead
	maxDiffSize = 50000

	// maxConventionsSize is the maximum size of conventions to include
	maxConventionsSize = 10000
)

// BuildPrompt constructs a comprehensive prompt for AI-powered PR description generation.
// The prompt includes all relevant context from PRContext formatted for Claude/Cursor API style.
func BuildPrompt(prContext *PRContext) string {
	var sections []string

	// Header
	sections = append(sections, "You are helping to generate a pull request description. Use the following context to create a comprehensive PR description.")

	// Branch information
	sections = append(sections, buildBranchSection(prContext))

	// Commit messages
	if len(prContext.CommitMessages) > 0 {
		sections = append(sections, buildCommitSection(prContext.CommitMessages))
	}

	// Changed files
	if len(prContext.ChangedFiles) > 0 {
		sections = append(sections, buildChangedFilesSection(prContext.ChangedFiles))
	}

	// Code diff
	if prContext.CodeDiff != "" {
		sections = append(sections, buildDiffSection(prContext.CodeDiff))
	}

	// Stack context
	if prContext.ParentPRInfo != nil || prContext.ChildPRInfo != nil || len(prContext.RelatedPRs) > 0 {
		sections = append(sections, buildStackSection(prContext))
	}

	// Project conventions
	if prContext.ProjectConventions != "" {
		sections = append(sections, buildConventionsSection(prContext.ProjectConventions))
	}

	// Output format instructions
	sections = append(sections, buildOutputFormatSection())

	return strings.Join(sections, "\n\n")
}

// buildBranchSection formats branch information
func buildBranchSection(prContext *PRContext) string {
	var lines []string
	lines = append(lines, "## Branch Information")
	lines = append(lines, fmt.Sprintf("- **Branch**: %s", prContext.BranchName))
	if prContext.ParentBranchName != "" {
		lines = append(lines, fmt.Sprintf("- **Parent Branch**: %s", prContext.ParentBranchName))
	}
	if prContext.TrunkBranchName != "" {
		lines = append(lines, fmt.Sprintf("- **Trunk Branch**: %s", prContext.TrunkBranchName))
	}
	return strings.Join(lines, "\n")
}

// buildCommitSection formats commit messages
func buildCommitSection(commitMessages []string) string {
	lines := make([]string, 0, len(commitMessages)+2)
	lines = append(lines, "## Commit Messages")
	lines = append(lines, "")
	for i, msg := range commitMessages {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, msg))
	}
	return strings.Join(lines, "\n")
}

// buildChangedFilesSection formats changed files with categorization
func buildChangedFilesSection(changedFiles []string) string {
	var lines []string
	lines = append(lines, "## Changed Files")
	lines = append(lines, "")

	// Categorize files
	var categories = map[string][]string{
		"Go":     {},
		"Tests":  {},
		"Config": {},
		"Docs":   {},
		"Other":  {},
	}

	for _, file := range changedFiles {
		category := categorizeFile(file)
		categories[category] = append(categories[category], file)
	}

	// Output categorized files
	for category, files := range categories {
		if len(files) > 0 {
			lines = append(lines, fmt.Sprintf("### %s", category))
			for _, file := range files {
				lines = append(lines, fmt.Sprintf("- %s", file))
			}
			lines = append(lines, "")
		}
	}

	return strings.TrimSuffix(strings.Join(lines, "\n"), "\n")
}

// categorizeFile categorizes a file by its extension/path
func categorizeFile(file string) string {
	lower := strings.ToLower(file)
	if strings.HasSuffix(lower, "_test.go") || strings.Contains(lower, "/test") {
		return "Tests"
	}
	if strings.HasSuffix(lower, ".go") {
		return "Go"
	}
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") {
		return "Docs"
	}
	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".toml") {
		return "Config"
	}
	return "Other"
}

// buildDiffSection formats code diff, truncating if too large
func buildDiffSection(diff string) string {
	var lines []string
	lines = append(lines, "## Code Diff")
	lines = append(lines, "")

	if len(diff) > maxDiffSize {
		// Include summary and first/last portions
		lines = append(lines, fmt.Sprintf("_Diff is large (%d characters). Showing summary and excerpts._", len(diff)))
		lines = append(lines, "")
		lines = append(lines, "### Beginning of diff:")
		lines = append(lines, "```")
		lines = append(lines, diff[:maxDiffSize/3])
		lines = append(lines, "```")
		lines = append(lines, "")
		lines = append(lines, "... (diff truncated) ...")
		lines = append(lines, "")
		lines = append(lines, "### End of diff:")
		lines = append(lines, "```")
		lines = append(lines, diff[len(diff)-maxDiffSize/3:])
		lines = append(lines, "```")
	} else {
		lines = append(lines, "```")
		lines = append(lines, diff)
		lines = append(lines, "```")
	}

	return strings.Join(lines, "\n")
}

// buildStackSection formats stack context (parent, child, related PRs)
func buildStackSection(prContext *PRContext) string {
	var lines []string
	lines = append(lines, "## Stack Context")

	// Parent PR
	if prContext.ParentPRInfo != nil {
		lines = append(lines, "")
		lines = append(lines, "### Parent PR")
		parentNum := prNumberNA
		if prContext.ParentPRInfo.Number != nil {
			parentNum = fmt.Sprintf("#%d", *prContext.ParentPRInfo.Number)
		}
		lines = append(lines, fmt.Sprintf("- **%s**: %s", parentNum, prContext.ParentPRInfo.Title))
		if prContext.ParentPRInfo.URL != "" {
			lines = append(lines, fmt.Sprintf("  - URL: %s", prContext.ParentPRInfo.URL))
		}
	}

	// Child PR
	if prContext.ChildPRInfo != nil {
		lines = append(lines, "")
		lines = append(lines, "### Child PR")
		childNum := prNumberNA
		if prContext.ChildPRInfo.Number != nil {
			childNum = fmt.Sprintf("#%d", *prContext.ChildPRInfo.Number)
		}
		lines = append(lines, fmt.Sprintf("- **%s**: %s", childNum, prContext.ChildPRInfo.Title))
		if prContext.ChildPRInfo.URL != "" {
			lines = append(lines, fmt.Sprintf("  - URL: %s", prContext.ChildPRInfo.URL))
		}
	}

	// Related PRs
	if len(prContext.RelatedPRs) > 0 {
		lines = append(lines, "")
		lines = append(lines, "### Related PRs in Stack")
		for _, related := range prContext.RelatedPRs {
			prNum := prNumberNA
			if related.Number != 0 {
				prNum = fmt.Sprintf("#%d", related.Number)
			}
			lines = append(lines, fmt.Sprintf("- **%s** (%s): %s", prNum, related.BranchName, related.Title))
			if related.URL != "" {
				lines = append(lines, fmt.Sprintf("  - URL: %s", related.URL))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// buildConventionsSection formats project conventions, truncating if too large
func buildConventionsSection(conventions string) string {
	var lines []string
	lines = append(lines, "## Project Conventions")
	lines = append(lines, "")

	if len(conventions) > maxConventionsSize {
		lines = append(lines, fmt.Sprintf("_Conventions truncated (showing first %d characters of %d total)_", maxConventionsSize, len(conventions)))
		lines = append(lines, "")
		lines = append(lines, conventions[:maxConventionsSize])
		lines = append(lines, "")
		lines = append(lines, "... (truncated) ...")
	} else {
		lines = append(lines, conventions)
	}

	return strings.Join(lines, "\n")
}

// BuildStackSuggestionPrompt creates a prompt for stack suggestion generation
func BuildStackSuggestionPrompt(diff string) string {
	// Truncate diff if too long
	maxDiffLength := 20000 // More context for analysis
	if len(diff) > maxDiffLength {
		diff = diff[:maxDiffLength] + "\n... (diff truncated)"
	}

	return fmt.Sprintf(`Analyze the following git diff and suggest how it should be split into a "stack" of logical branches (stacked changes).

Each branch in the stack should represent a single, reviewable unit of work. Consider:
- Logical dependencies (refactors before new features)
- Domain boundaries (backend vs frontend, data model vs UI)
- Reviewability (keep branches focused and manageable in size)
- Testing boundaries

Output your suggestion in the following YAML format:

- branch: <branch-name>
  files:
    - <file1>
    - <file2>
  rationale: <brief explanation of why these changes are grouped together>
  message: <suggested conventional commit message for this layer>

Rules:
- Use branch names that are descriptive and hyphenated (e.g., "refactor-user-model").
- Use conventional commit format for messages (e.g., "feat: add user preferences").
- Every file changed in the diff must be assigned to at least one branch.
- Return ONLY the YAML structure, no additional text, explanations, or markdown formatting (unless specifically asked for code blocks).
- Order branches from bottom to top (base dependencies first).
- DO NOT use markdown code blocks in your response.

Git diff:
%s`, diff)
}

// buildOutputFormatSection provides instructions for structured output
func buildOutputFormatSection() string {
	return `## Output Format

Please provide a structured PR description with the following sections:

### Title
A concise PR title (50-72 characters recommended). Should be clear and descriptive.

### Summary
A brief overview of the changes (2-4 sentences). Explain what this PR does and why.

### Details
Optional detailed explanation of the changes, implementation approach, or important considerations.

### Testing Instructions
If applicable, provide instructions for testing the changes. Include any setup steps, test cases, or verification steps.

### Related PRs
If there are related PRs in the stack (parent, child, or siblings), format them as markdown links:
- Depends on: [PR Title](#123)
- Blocked by: [PR Title](#456)

Format your response as:
---
TITLE: <title here>
---
BODY:
<full body here>`
}

// ParseAIResponse parses the AI response to extract title and body.
// Expected format:
//
//	---
//	TITLE: <title>
//	---
//	BODY:
//	<body content>
//
// If the format doesn't match, it attempts to extract title from first line
// and uses the rest as body.
func ParseAIResponse(response string) (title string, body string, err error) {
	response = strings.TrimSpace(response)

	if response == "" {
		return "", "", fmt.Errorf("failed to parse AI response: empty response")
	}

	// Try to parse structured format
	if strings.HasPrefix(response, "---") {
		// Find the sections between "---" markers
		parts := strings.SplitN(response, "---", 3)
		if len(parts) >= 3 {
			// parts[0] is before first "---" (should be empty or whitespace)
			// parts[1] is between first and second "---" (title section)
			// parts[2] is after second "---" (body section)

			// Extract title
			titleSection := strings.TrimSpace(parts[1])
			if strings.HasPrefix(titleSection, "TITLE:") {
				title = strings.TrimSpace(strings.TrimPrefix(titleSection, "TITLE:"))
			} else if titleSection != "" {
				// Title might be on its own line
				title = titleSection
			}

			// Extract body
			bodySection := parts[2] // Don't trim yet, preserve structure
			// Look for "BODY:" marker (may be on its own line)
			if idx := strings.Index(bodySection, "BODY:"); idx >= 0 {
				// Get everything after "BODY:"
				body = bodySection[idx+5:]
				// Trim leading whitespace, colons, and newlines
				body = strings.TrimLeft(body, " :\n\r\t")
			} else {
				// No BODY: marker, use the whole section
				body = strings.TrimSpace(bodySection)
			}

			if title != "" {
				return title, body, nil
			}
		}
	}

	// Fallback: if title is empty, try to extract from first line
	if title == "" {
		lines := strings.Split(response, "\n")
		if len(lines) > 0 {
			firstLine := strings.TrimSpace(lines[0])
			// Remove common prefixes
			firstLine = strings.TrimPrefix(firstLine, "TITLE:")
			firstLine = strings.TrimPrefix(firstLine, "Title:")
			firstLine = strings.TrimSpace(firstLine)

			// Handle long titles - truncate at word boundary
			if len(firstLine) > 80 {
				// Look for last space in first 80 chars
				if lastSpace := strings.LastIndex(firstLine[:80], " "); lastSpace >= 50 {
					title = firstLine[:lastSpace]
					remaining := firstLine[lastSpace:] + "\n" + strings.Join(lines[1:], "\n")
					body = strings.TrimSpace(remaining)
				} else {
					// No good word boundary, truncate at 72
					title = firstLine[:72]
					remaining := firstLine[72:] + "\n" + strings.Join(lines[1:], "\n")
					body = strings.TrimSpace(remaining)
				}
			} else {
				title = firstLine
				// Use rest as body
				if len(lines) > 1 {
					body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
				}
			}
		}
	}

	// Final fallback: use first line or first 72 chars as title, rest as body
	if title == "" {
		// Single line response
		if len(response) <= 72 {
			title = response
			body = ""
		} else {
			// Try to break at word boundary (prefer around 72, but allow up to 80)
			searchLen := 80
			if len(response) < searchLen {
				searchLen = len(response)
			}
			if lastSpace := strings.LastIndex(response[:searchLen], " "); lastSpace >= 50 {
				title = response[:lastSpace]
				body = strings.TrimSpace(response[lastSpace:])
			} else {
				// No good word boundary, truncate at 72
				title = response[:72]
				body = strings.TrimSpace(response[72:])
			}
		}
	}

	// Ensure we have at least a title
	if title == "" {
		return "", "", fmt.Errorf("failed to parse AI response: no title found")
	}

	return title, body, nil
}
