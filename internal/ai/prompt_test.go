package ai

import (
	"strings"
	"testing"

	"stackit.dev/stackit/internal/engine"
)

const testTitleAddLogin = "Add login functionality"

func TestBuildPrompt_Basic(t *testing.T) {
	prCtx := &PRContext{
		BranchName:       "feature/add-login",
		ParentBranchName: "main",
		TrunkBranchName:  "main",
		CommitMessages:   []string{testTitleAddLogin, "Fix typo in login"},
		ChangedFiles:     []string{"internal/auth/login.go", "internal/auth/login_test.go"},
		CodeDiff:         "diff --git a/login.go...",
	}

	prompt := BuildPrompt(prCtx)

	// Check that all sections are present
	if !strings.Contains(prompt, "Branch Information") {
		t.Error("Prompt missing branch information section")
	}
	if !strings.Contains(prompt, "Commit Messages") {
		t.Error("Prompt missing commit messages section")
	}
	if !strings.Contains(prompt, "Changed Files") {
		t.Error("Prompt missing changed files section")
	}
	if !strings.Contains(prompt, "Code Diff") {
		t.Error("Prompt missing code diff section")
	}
	if !strings.Contains(prompt, "Output Format") {
		t.Error("Prompt missing output format section")
	}

	// Check branch info
	if !strings.Contains(prompt, "feature/add-login") {
		t.Error("Prompt missing branch name")
	}
	if !strings.Contains(prompt, "main") {
		t.Error("Prompt missing parent/trunk branch")
	}

	// Check commit messages
	if !strings.Contains(prompt, testTitleAddLogin) {
		t.Error("Prompt missing commit message")
	}
}

func TestBuildPrompt_EmptyContext(t *testing.T) {
	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	prompt := BuildPrompt(prCtx)

	// Should still have branch info and output format
	if !strings.Contains(prompt, "Branch Information") {
		t.Error("Prompt missing branch information section")
	}
	if !strings.Contains(prompt, "Output Format") {
		t.Error("Prompt missing output format section")
	}

	// Should not have empty sections
	if strings.Contains(prompt, "Commit Messages\n\n") {
		t.Error("Prompt should not have empty commit messages section")
	}
}

func TestBuildPrompt_StackContext(t *testing.T) {
	parentNum := 123
	childNum := 456
	prCtx := &PRContext{
		BranchName:       "feature/middle",
		ParentBranchName: "feature/base",
		ParentPRInfo: &engine.PrInfo{
			Number: &parentNum,
			Title:  "Base feature",
			URL:    "https://github.com/example/repo/pull/123",
		},
		ChildPRInfo: &engine.PrInfo{
			Number: &childNum,
			Title:  "Child feature",
			URL:    "https://github.com/example/repo/pull/456",
		},
		RelatedPRs: []RelatedPR{
			{
				BranchName: "feature/sibling",
				Title:      "Sibling PR",
				URL:        "https://github.com/example/repo/pull/789",
				Number:     789,
			},
		},
	}

	prompt := BuildPrompt(prCtx)

	// Check stack context
	if !strings.Contains(prompt, "Stack Context") {
		t.Error("Prompt missing stack context section")
	}
	if !strings.Contains(prompt, "Parent PR") {
		t.Error("Prompt missing parent PR section")
	}
	if !strings.Contains(prompt, "Child PR") {
		t.Error("Prompt missing child PR section")
	}
	if !strings.Contains(prompt, "Related PRs in Stack") {
		t.Error("Prompt missing related PRs section")
	}

	// Check PR info
	if !strings.Contains(prompt, "#123") {
		t.Error("Prompt missing parent PR number")
	}
	if !strings.Contains(prompt, "#456") {
		t.Error("Prompt missing child PR number")
	}
	if !strings.Contains(prompt, "#789") {
		t.Error("Prompt missing related PR number")
	}
}

func TestBuildPrompt_LargeDiff(t *testing.T) {
	largeDiff := strings.Repeat("diff line\n", 10000) // ~100KB
	prCtx := &PRContext{
		BranchName: "feature/large-change",
		CodeDiff:   largeDiff,
	}

	prompt := BuildPrompt(prCtx)

	// Should truncate and show summary
	if !strings.Contains(prompt, "Diff is large") {
		t.Error("Prompt should indicate large diff truncation")
	}
	if !strings.Contains(prompt, "Beginning of diff") {
		t.Error("Prompt should show beginning of diff")
	}
	if !strings.Contains(prompt, "End of diff") {
		t.Error("Prompt should show end of diff")
	}
}

func TestBuildPrompt_FileCategorization(t *testing.T) {
	prCtx := &PRContext{
		BranchName: "feature/test",
		ChangedFiles: []string{
			"internal/auth/login.go",
			"internal/auth/login_test.go",
			"README.md",
			"config.yml",
			"scripts/setup.sh",
		},
	}

	prompt := BuildPrompt(prCtx)

	// Check categorization
	if !strings.Contains(prompt, "### Go") {
		t.Error("Prompt missing Go category")
	}
	if !strings.Contains(prompt, "### Tests") {
		t.Error("Prompt missing Tests category")
	}
	if !strings.Contains(prompt, "### Docs") {
		t.Error("Prompt missing Docs category")
	}
	if !strings.Contains(prompt, "### Config") {
		t.Error("Prompt missing Config category")
	}

	// Check files in categories
	if !strings.Contains(prompt, "login.go") {
		t.Error("Prompt missing Go file")
	}
	if !strings.Contains(prompt, "login_test.go") {
		t.Error("Prompt missing test file")
	}
}

func TestParseAIResponse_StructuredFormat(t *testing.T) {
	response := `---
TITLE: ` + testTitleAddLogin + `
---
BODY:
## Summary
This PR adds user login functionality.

## Details
- Implemented authentication
- Added tests

## Related PRs
- Depends on: [Base PR](#123)`

	title, body, err := ParseAIResponse(response)
	if err != nil {
		t.Fatalf("ParseAIResponse failed: %v", err)
	}

	if title != testTitleAddLogin {
		t.Errorf("Expected title '%s', got '%s'", testTitleAddLogin, title)
	}

	if !strings.Contains(body, "Summary") {
		t.Error("Body missing summary section")
	}
	if !strings.Contains(body, "adds user login functionality") {
		t.Error("Body should contain summary content")
	}
}

func TestParseAIResponse_SimpleFormat(t *testing.T) {
	response := testTitleAddLogin + `

This PR adds user login functionality with authentication.`

	title, body, err := ParseAIResponse(response)
	if err != nil {
		t.Fatalf("ParseAIResponse failed: %v", err)
	}

	if title != testTitleAddLogin {
		t.Errorf("Expected title '%s', got '%s'", testTitleAddLogin, title)
	}

	if !strings.Contains(body, "This PR adds") {
		t.Error("Body missing content")
	}
}

func TestParseAIResponse_WithTitlePrefix(t *testing.T) {
	response := "TITLE: " + testTitleAddLogin + `

This is the body content.`

	title, body, err := ParseAIResponse(response)
	if err != nil {
		t.Fatalf("ParseAIResponse failed: %v", err)
	}

	if title != testTitleAddLogin {
		t.Errorf("Expected title '%s', got '%s'", testTitleAddLogin, title)
	}

	if !strings.Contains(body, "This is the body") {
		t.Error("Body missing content")
	}
}

func TestParseAIResponse_LongTitle(t *testing.T) {
	// Response with title longer than 72 chars
	longTitle := strings.Repeat("word ", 20) // ~100 chars
	response := longTitle + "\n\nBody content here."

	title, body, err := ParseAIResponse(response)
	if err != nil {
		t.Fatalf("ParseAIResponse failed: %v", err)
	}

	// Should truncate at word boundary
	if len(title) > 80 {
		t.Errorf("Title too long: %d chars", len(title))
	}

	if !strings.Contains(body, "Body content") {
		t.Error("Body missing content")
	}
}

func TestParseAIResponse_EmptyResponse(t *testing.T) {
	_, _, err := ParseAIResponse("")
	if err == nil {
		t.Error("Expected error for empty response")
	}
}

func TestParseAIResponse_WhitespaceOnly(t *testing.T) {
	_, _, err := ParseAIResponse("   \n\t  ")
	if err == nil {
		t.Error("Expected error for whitespace-only response")
	}
}
