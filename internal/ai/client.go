// Package ai provides AI-powered features for stackit, including context collection
// for PR description generation.
package ai

import (
	"context"
)

// AIClient defines the interface for AI-powered PR description generation.
// Implementations should use the provided PRContext to generate a title and body
// for a pull request description.
//
// The GeneratePRDescription method should:
// - Use the PRContext to build a comprehensive prompt
// - Call the AI service (e.g., Claude/Cursor API)
// - Parse the response to extract title and body
// - Return structured output suitable for PR creation
//
// Implementations may handle rate limiting, retries, and error handling
// as appropriate for their specific AI service.
type AIClient interface {
	// GeneratePRDescription generates a PR title and body from the provided context.
	// The context parameter is used for cancellation and timeout handling.
	// The prContext contains all necessary information about the branch, commits, diffs, etc.
	//
	// Returns:
	//   - title: A concise PR title (typically 50-72 characters)
	//   - body: A formatted PR body with summary, details, and related PRs
	//   - err: Any error that occurred during generation or parsing
	GeneratePRDescription(ctx context.Context, prContext *PRContext) (title string, body string, err error)
}
