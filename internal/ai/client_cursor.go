package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CursorAgentClient implements Client using cursor-agent CLI
type CursorAgentClient struct{}

// NewCursorAgentClient creates a new CursorAgentClient
func NewCursorAgentClient() (*CursorAgentClient, error) {
	// Check if cursor-agent CLI is available
	if !isCursorAgentAvailable() {
		return nil, fmt.Errorf("cursor-agent CLI not available in PATH")
	}

	return &CursorAgentClient{}, nil
}

// isCursorAgentAvailable checks if cursor-agent CLI is available
func isCursorAgentAvailable() bool {
	cmd := exec.Command("cursor-agent", "--version")
	err := cmd.Run()
	return err == nil
}

// GenerateCommitMessage generates a commit message from staged changes using cursor-agent
func (c *CursorAgentClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	// Build prompt for commit message generation
	prompt := BuildCommitMessagePrompt(diff)

	// Use cursor-agent CLI
	message, err := c.callCursorAgentCLI(ctx, prompt)

	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Clean and validate the message
	message = strings.TrimSpace(message)
	if message == "" {
		return "", fmt.Errorf("generated empty commit message")
	}

	// Ensure it follows conventional commit format if possible
	message = ensureConventionalCommitFormat(message)

	return message, nil
}

// GeneratePRDescription generates a PR title and body from the provided context
func (c *CursorAgentClient) GeneratePRDescription(ctx context.Context, prContext *PRContext) (string, string, error) {
	// Build prompt for PR description generation
	prompt := BuildPrompt(prContext)

	// Use cursor-agent CLI
	response, err := c.callCursorAgentCLI(ctx, prompt)

	if err != nil {
		return "", "", fmt.Errorf("failed to generate PR description: %w", err)
	}

	// Parse response to extract title and body
	title, body := parsePRResponse(response)
	return title, body, nil
}

// GenerateStackSuggestion suggests a stack structure for staged changes using cursor-agent
func (c *CursorAgentClient) GenerateStackSuggestion(ctx context.Context, diff string) (*StackSuggestion, error) {
	// Build prompt for stack suggestion generation
	prompt := BuildStackSuggestionPrompt(diff)

	// Use cursor-agent CLI
	response, err := c.callCursorAgentCLI(ctx, prompt)

	if err != nil {
		return nil, fmt.Errorf("failed to generate stack suggestion: %w", err)
	}

	// Parse YAML-like response
	suggestion, err := parseStackSuggestion(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stack suggestion: %w", err)
	}

	return suggestion, nil
}

// callCursorAgentCLI calls cursor-agent CLI tool
func (c *CursorAgentClient) callCursorAgentCLI(ctx context.Context, prompt string) (string, error) {
	// Use cursor-agent CLI with the prompt
	// The -p flag runs in non-interactive mode
	cmd := exec.CommandContext(ctx, "cursor-agent", "-p", prompt)

	// Log the command being executed (truncate prompt for readability)
	promptPreview := prompt
	if len(promptPreview) > 200 {
		promptPreview = promptPreview[:200] + "... (truncated)"
	}
	_, _ = fmt.Fprintf(os.Stderr, "Running cursor-agent command: cursor-agent -p %q\n", promptPreview)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if cursor-agent is not found
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return "", fmt.Errorf("cursor-agent not found in PATH")
		}

		// Capture full output for debugging
		stdoutStr := stdout.String()
		stderrStr := stderr.String()

		// Build detailed error message
		var errorMsg strings.Builder
		errorMsg.WriteString("cursor-agent failed with exit code")
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			errorMsg.WriteString(fmt.Sprintf(" %d", exitError.ExitCode()))
		}
		errorMsg.WriteString(fmt.Sprintf(": %v\n", err))

		if stderrStr != "" {
			errorMsg.WriteString("\nstderr:\n")
			errorMsg.WriteString(stderrStr)
		}

		if stdoutStr != "" {
			errorMsg.WriteString("\nstdout:\n")
			errorMsg.WriteString(stdoutStr)
		}

		// Also log to stderr for immediate visibility
		_, _ = fmt.Fprintf(os.Stderr, "\n=== cursor-agent error ===\n%s\n", errorMsg.String())

		return "", fmt.Errorf("%s", errorMsg.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("cursor-agent returned empty output")
	}

	// Strip markdown code blocks if present
	output = stripMarkdownCodeBlocks(output)

	// Extract just the commit message from the output
	// cursor-agent might return additional formatting, so we clean it up
	lines := strings.Split(output, "\n")
	messageLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and common prefixes
		if line == "" || strings.HasPrefix(line, "Generated:") || strings.HasPrefix(line, "Commit message:") {
			continue
		}
		// Remove prefix if present
		line = strings.TrimPrefix(line, "Generated: ")
		line = strings.TrimPrefix(line, "Commit message: ")
		messageLines = append(messageLines, line)
	}

	result := strings.Join(messageLines, "\n")
	if result == "" {
		result = output // Fall back to raw output if cleaning removed everything
	}

	// Take only the first line (subject line) - commit messages should be single line
	firstLine := strings.Split(result, "\n")[0]
	firstLine = strings.TrimSpace(firstLine)

	// If the first line is too long, truncate it (but try to preserve the type: prefix)
	if len(firstLine) > 72 {
		// Try to preserve "type: " prefix if present
		if colonIdx := strings.Index(firstLine, ":"); colonIdx > 0 && colonIdx < 20 {
			prefix := firstLine[:colonIdx+1]
			description := firstLine[colonIdx+1:]
			description = strings.TrimSpace(description)
			// Truncate description to fit in 72 chars total
			maxDescLen := 72 - len(prefix) - 1 // -1 for space
			if len(description) > maxDescLen {
				description = description[:maxDescLen-3] + "..."
			}
			firstLine = prefix + " " + description
		} else {
			// No type prefix, just truncate
			firstLine = firstLine[:69] + "..."
		}
	}

	return firstLine, nil
}

// BuildCommitMessagePrompt creates a prompt for commit message generation
// This is exported so it can be used by debug commands
func BuildCommitMessagePrompt(diff string) string {
	// Truncate diff if too long (cursor-agent might have limits)
	maxDiffLength := 10000
	if len(diff) > maxDiffLength {
		diff = diff[:maxDiffLength] + "\n... (diff truncated)"
	}

	return fmt.Sprintf(`Generate a commit message for the following git diff. Follow Conventional Commits format: <type>[optional scope]: <description>

Requirements:
- Use appropriate type: feat, fix, docs, style, refactor, perf, test, chore, or ci
- Keep it SHORT and concise (50-72 characters total, single line only)
- Use imperative mood (e.g., "add feature" not "added feature" or "adds feature")
- Accurately describe the changes in the diff
- Return ONLY a single line of plain text, no markdown formatting, no code blocks, no quotes, no line breaks, no additional text

Git diff:
%s

Return only a single line commit message in the format: <type>[optional scope]: <short description>`, diff)
}

// stripMarkdownCodeBlocks removes markdown code blocks from the output
func stripMarkdownCodeBlocks(text string) string {
	// Remove triple backtick code blocks
	text = strings.TrimSpace(text)

	// Remove opening ```language or just ```
	if strings.HasPrefix(text, "```") {
		// Find the first newline after ```
		firstNewline := strings.Index(text, "\n")
		if firstNewline > 0 {
			text = text[firstNewline+1:]
		} else {
			// No newline, just remove the ```
			text = strings.TrimPrefix(text, "```")
		}
	}

	// Remove closing ```
	text = strings.TrimSuffix(text, "```")

	// Also handle single backticks that might wrap the entire message
	text = strings.Trim(text, "`")

	return strings.TrimSpace(text)
}

// ensureConventionalCommitFormat ensures the message follows conventional commit format
func ensureConventionalCommitFormat(message string) string {
	message = strings.TrimSpace(message)

	// Remove any leading/trailing quotes
	message = strings.Trim(message, `"'`)

	// Check if it already starts with a conventional commit type
	types := []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore", "ci"}
	for _, t := range types {
		if strings.HasPrefix(message, t+":") || strings.HasPrefix(message, t+"(") {
			return message
		}
	}

	// If it doesn't start with a type, try to infer and prepend
	// For now, default to "feat" if we can't determine
	lower := strings.ToLower(message)
	if strings.Contains(lower, "fix") || strings.Contains(lower, "bug") || strings.Contains(lower, "error") {
		if !strings.HasPrefix(message, "fix:") {
			message = "fix: " + message
		}
	} else {
		// Default to feat if no clear type
		if !strings.HasPrefix(message, "feat:") {
			message = "feat: " + message
		}
	}

	return message
}

// parsePRResponse parses the API response to extract PR title and body
func parsePRResponse(response string) (string, string) {
	response = strings.TrimSpace(response)

	// Try to split on common separators
	lines := strings.Split(response, "\n")
	if len(lines) == 0 {
		return response, ""
	}

	// First line is typically the title
	title := strings.TrimSpace(lines[0])

	// Remove title markers if present
	title = strings.TrimPrefix(title, "Title: ")
	title = strings.TrimPrefix(title, "# ")

	// Rest is the body
	body := strings.TrimSpace(strings.Join(lines[1:], "\n"))

	// Remove body markers if present
	body = strings.TrimPrefix(body, "Body: ")
	body = strings.TrimPrefix(body, "Description: ")

	return title, body
}

// parseStackSuggestion parses the YAML-like response from AI
func parseStackSuggestion(response string) (*StackSuggestion, error) {
	response = stripMarkdownCodeBlocks(response)
	lines := strings.Split(response, "\n")
	suggestion := &StackSuggestion{Layers: []StackLayer{}}

	var currentLayer *StackLayer
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// New branch starts with "- branch:" or "branch:"
		if strings.HasPrefix(trimmed, "- branch:") || (strings.HasPrefix(trimmed, "branch:") && currentLayer == nil) {
			if currentLayer != nil {
				suggestion.Layers = append(suggestion.Layers, *currentLayer)
			}
			branchName := ""
			if strings.HasPrefix(trimmed, "- branch:") {
				branchName = strings.TrimSpace(strings.TrimPrefix(trimmed, "- branch:"))
			} else {
				branchName = strings.TrimSpace(strings.TrimPrefix(trimmed, "branch:"))
			}
			branchName = strings.Trim(branchName, `"'`)
			currentLayer = &StackLayer{BranchName: branchName}
			continue
		}

		if currentLayer == nil {
			continue
		}

		// Files: "files: [file1, file2]" or "files:" or "- files:"
		if strings.HasPrefix(trimmed, "files:") || strings.HasPrefix(trimmed, "- files:") {
			filesStr := ""
			if strings.HasPrefix(trimmed, "files:") {
				filesStr = strings.TrimSpace(strings.TrimPrefix(trimmed, "files:"))
			} else {
				filesStr = strings.TrimSpace(strings.TrimPrefix(trimmed, "- files:"))
			}

			if strings.HasPrefix(filesStr, "[") && strings.HasSuffix(filesStr, "]") {
				filesStr = strings.Trim(filesStr, "[]")
				parts := strings.Split(filesStr, ",")
				for _, p := range parts {
					file := strings.TrimSpace(p)
					file = strings.Trim(file, `"'`)
					if file != "" {
						currentLayer.Files = append(currentLayer.Files, file)
					}
				}
			}
			continue
		}

		// Rationale: "rationale: ..." or "- rationale: ..."
		if strings.HasPrefix(trimmed, "rationale:") || strings.HasPrefix(trimmed, "- rationale:") {
			rationale := ""
			if strings.HasPrefix(trimmed, "rationale:") {
				rationale = strings.TrimSpace(strings.TrimPrefix(trimmed, "rationale:"))
			} else {
				rationale = strings.TrimSpace(strings.TrimPrefix(trimmed, "- rationale:"))
			}
			currentLayer.Rationale = strings.Trim(rationale, `"'`)
			continue
		}

		// Message: "message: ..." or "commit: ..." or "- message: ..."
		if strings.HasPrefix(trimmed, "message:") || strings.HasPrefix(trimmed, "commit:") || strings.HasPrefix(trimmed, "- message:") || strings.HasPrefix(trimmed, "- commit:") {
			message := ""
			switch {
			case strings.HasPrefix(trimmed, "message:"):
				message = strings.TrimSpace(strings.TrimPrefix(trimmed, "message:"))
			case strings.HasPrefix(trimmed, "commit:"):
				message = strings.TrimSpace(strings.TrimPrefix(trimmed, "commit:"))
			case strings.HasPrefix(trimmed, "- message:"):
				message = strings.TrimSpace(strings.TrimPrefix(trimmed, "- message:"))
			default:
				message = strings.TrimSpace(strings.TrimPrefix(trimmed, "- commit:"))
			}
			currentLayer.CommitMessage = strings.Trim(message, `"'`)
			continue
		}

		// Handle list items for files if they are on separate lines
		if strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "- branch:") {
			file := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			file = strings.Trim(file, `"'`)
			if file != "" {
				// Avoid adding rationale/message as files if they happen to start with "- "
				if !strings.HasPrefix(file, "rationale:") && !strings.HasPrefix(file, "message:") && !strings.HasPrefix(file, "commit:") {
					currentLayer.Files = append(currentLayer.Files, file)
				}
			}
		}
	}

	if currentLayer != nil {
		suggestion.Layers = append(suggestion.Layers, *currentLayer)
	}

	if len(suggestion.Layers) == 0 {
		return nil, fmt.Errorf("no valid stack layers found in AI response")
	}

	return suggestion, nil
}
