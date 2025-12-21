// Package ai provides AI-powered features for stackit, including context collection
// for PR description generation.
package ai

import (
	"context"
)

// StackLayer represents a layer in a proposed stack of branches.
type StackLayer struct {
	BranchName    string
	Files         []string
	Rationale     string
	CommitMessage string
}

// StackSuggestion represents a proposed stack structure for changes.
type StackSuggestion struct {
	Layers []StackLayer
}

// Client defines the interface for AI-powered PR description generation.
type Client interface {
	// GeneratePRDescription generates a PR title and body from the provided context.
	// The context parameter is used for cancellation and timeout handling.
	// The prContext contains all necessary information about the branch, commits, diffs, etc.
	//
	// Returns:
	//   - title: A concise PR title (typically 50-72 characters)
	//   - body: A formatted PR body with summary, details, and related PRs
	//   - err: Any error that occurred during generation or parsing
	GeneratePRDescription(ctx context.Context, prContext *PRContext) (title string, body string, err error)

	// GenerateCommitMessage generates a commit message from staged changes.
	// The context parameter is used for cancellation and timeout handling.
	// The diff contains the staged changes to analyze.
	//
	// Returns:
	//   - message: A commit message following conventional commit format (e.g., "feat: add feature")
	//   - err: Any error that occurred during generation or parsing
	GenerateCommitMessage(ctx context.Context, diff string) (message string, err error)

	// GenerateStackSuggestion suggests a stack structure for staged changes.
	// The context parameter is used for cancellation and timeout handling.
	// The diff contains the staged changes to analyze.
	//
	// Returns:
	//   - suggestion: A structured stack suggestion
	//   - err: Any error that occurred during generation or parsing
	GenerateStackSuggestion(ctx context.Context, diff string) (suggestion *StackSuggestion, err error)
}
