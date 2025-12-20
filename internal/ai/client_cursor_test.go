package ai

import (
	"strings"
	"testing"
)

// NOTE: These tests do NOT make real API calls or invoke cursor-agent CLI.
// They only test:
// - Client creation logic (struct initialization)
// - Helper functions (prompt building, formatting, parsing)
//
// Tests that would require real API calls (GenerateCommitMessage, GeneratePRDescription)
// are not included here to avoid dependencies on external binaries.

func TestNewCursorAgentClient(t *testing.T) {
	// These tests only verify client creation logic, not actual CLI calls
	t.Run("creates client when cursor-agent is available", func(t *testing.T) {
		// Since we can't easily mock isCursorAgentAvailable without refactoring,
		// and we know it's available in the environment where this test runs,
		// we just check if it works or fails gracefully.
		client, err := NewCursorAgentClient()
		if err != nil {
			// If it's not available in the test environment, that's fine too
			t.Logf("cursor-agent not available: %v", err)
			return
		}

		if client == nil {
			t.Fatal("Client is nil")
		}
	})
}

func TestBuildCommitMessagePrompt(t *testing.T) {
	diff := "diff --git a/file.go b/file.go\n+new code"
	prompt := BuildCommitMessagePrompt(diff)

	if prompt == "" {
		t.Fatal("Prompt should not be empty")
	}

	if !strings.Contains(prompt, diff) {
		t.Error("Prompt should contain the diff")
	}

	if !strings.Contains(prompt, "Conventional Commits") {
		t.Error("Prompt should mention Conventional Commits")
	}

	if !strings.Contains(prompt, "no markdown") {
		t.Error("Prompt should explicitly request no markdown formatting")
	}
}

func TestStripMarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no markdown",
			input:    "feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "with code block",
			input:    "```\nfeat: add new feature\n```",
			expected: "feat: add new feature",
		},
		{
			name:     "with language in code block",
			input:    "```text\nfeat: add new feature\n```",
			expected: "feat: add new feature",
		},
		{
			name:     "with single backticks",
			input:    "`feat: add new feature`",
			expected: "feat: add new feature",
		},
		{
			name:     "multiline with code block",
			input:    "```\nfix: enhance error reporting\nwith detailed output\n```",
			expected: "fix: enhance error reporting\nwith detailed output",
		},
		{
			name:     "code block with extra whitespace",
			input:    "```\n  feat: add feature  \n```",
			expected: "feat: add feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeBlocks(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEnsureConventionalCommitFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already formatted",
			input:    "feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "fix type",
			input:    "fix bug in code",
			expected: "fix: fix bug in code",
		},
		{
			name:     "no type",
			input:    "add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "with scope",
			input:    "feat(api): add endpoint",
			expected: "feat(api): add endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureConventionalCommitFormat(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParsePRResponse(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTitle string
		expectedBody  string
	}{
		{
			name:          "simple response",
			input:         "Title\nBody content",
			expectedTitle: "Title",
			expectedBody:  "Body content",
		},
		{
			name:          "with title prefix",
			input:         "Title: My Title\nBody",
			expectedTitle: "My Title",
			expectedBody:  "Body",
		},
		{
			name:          "with markdown header",
			input:         "# My Title\nBody content",
			expectedTitle: "My Title",
			expectedBody:  "Body content",
		},
		{
			name:          "single line",
			input:         "Just a title",
			expectedTitle: "Just a title",
			expectedBody:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body := parsePRResponse(tt.input)
			if title != tt.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.expectedTitle, title)
			}
			if body != tt.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}
