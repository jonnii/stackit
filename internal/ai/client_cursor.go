package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CursorAgentClient implements AIClient using cursor-agent CLI or Cursor API
type CursorAgentClient struct {
	useCLI     bool
	apiURL     string
	apiKey     string
	httpClient *http.Client
}

// CursorAgentOptions contains configuration for CursorAgentClient
type CursorAgentOptions struct {
	UseCLI  bool          // Use cursor-agent CLI instead of API (default: true if CLI available)
	APIURL  string        // Optional: custom API URL (defaults to Cursor API)
	APIKey  string        // Optional: API key (defaults to reading from environment)
	Timeout time.Duration // Optional: request timeout (defaults to 30s)
}

// NewCursorAgentClient creates a new CursorAgentClient
// If options are nil, uses defaults (tries CLI first, falls back to API)
func NewCursorAgentClient(opts *CursorAgentOptions) (*CursorAgentClient, error) {
	useCLI := true
	if opts != nil {
		useCLI = opts.UseCLI
	}

	// Check if cursor-agent CLI is available
	if useCLI {
		if !isCursorAgentAvailable() {
			useCLI = false
		}
	}

	var apiURL, apiKey string
	var httpClient *http.Client

	// If not using CLI, set up API client
	if !useCLI {
		apiURL = "https://api.cursor.com/v1/chat/completions"
		if opts != nil && opts.APIURL != "" {
			apiURL = opts.APIURL
		}

		apiKey = ""
		if opts != nil {
			apiKey = opts.APIKey
		}
		if apiKey == "" {
			apiKey = os.Getenv("CURSOR_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("CURSOR_API_KEY environment variable not set and cursor-agent CLI not available")
			}
		}

		timeout := 30 * time.Second
		if opts != nil && opts.Timeout > 0 {
			timeout = opts.Timeout
		}

		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	return &CursorAgentClient{
		useCLI:     useCLI,
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
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

	var message string
	var err error

	if c.useCLI {
		// Use cursor-agent CLI
		message, err = c.callCursorAgentCLI(ctx, prompt)
	} else {
		// Use Cursor API
		message, err = c.callCursorAPI(ctx, prompt)
	}

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

	var response string
	var err error

	if c.useCLI {
		// Use cursor-agent CLI
		response, err = c.callCursorAgentCLI(ctx, prompt)
	} else {
		// Use Cursor API
		response, err = c.callCursorAPI(ctx, prompt)
	}

	if err != nil {
		return "", "", fmt.Errorf("failed to generate PR description: %w", err)
	}

	// Parse response to extract title and body
	title, body := parsePRResponse(response)
	return title, body, nil
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
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return "", fmt.Errorf("cursor-agent not found in PATH. Install it or set CURSOR_API_KEY for API mode")
		}
		return "", fmt.Errorf("cursor-agent failed: %w, stderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("cursor-agent returned empty output")
	}

	// Extract just the commit message from the output
	// cursor-agent might return additional formatting, so we clean it up
	lines := strings.Split(output, "\n")
	var messageLines []string
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
		return output, nil // Fall back to raw output if cleaning removed everything
	}

	return result, nil
}

// callCursorAPI makes a request to the Cursor API
func (c *CursorAgentClient) callCursorAPI(ctx context.Context, prompt string) (string, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"model": "claude-3-5-sonnet-20241022", // Cursor's default model
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  1000,
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResponse.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices in API response")
	}

	return apiResponse.Choices[0].Message.Content, nil
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
- Keep it concise (under 72 characters for the first line)
- Accurately describe the changes

Git diff:
%s

Return only the commit message, no additional text.`, diff)
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
