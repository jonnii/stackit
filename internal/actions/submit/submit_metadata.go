// Package submit provides functionality for submitting stacked branches as pull requests.
package submit

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// GetPRTitle gets the PR title, prompting if needed
func GetPRTitle(branchName string, editInline bool, existingTitle string, ctx *runtime.Context) (string, error) {
	// First check if we have a saved title
	title := existingTitle
	if title == "" {
		// Otherwise, use the subject of the oldest commit on the branch
		subject, err := git.GetCommitSubject(ctx.Context, branchName)
		if err != nil {
			// Non-fatal, use branch name as fallback
			title = branchName
		} else {
			title = subject
		}
	}

	if !editInline {
		return title, nil
	}

	// Prompt for title
	result, err := tui.PromptTextInput("Title:", title)
	if err != nil {
		return "", fmt.Errorf("failed to get PR title: %w", err)
	}

	return result, nil
}

// GetPRBody gets the PR body, prompting if needed
func GetPRBody(branchName string, editInline bool, existingBody string, ctx *runtime.Context) (string, error) {
	body := existingBody
	if body == "" {
		// Infer from commit messages
		messages, err := git.GetCommitMessages(ctx.Context, branchName)
		if err == nil && len(messages) > 0 {
			if len(messages) == 1 {
				// Single commit - use body (skip first line which is subject)
				lines := strings.Split(messages[0], "\n")
				if len(lines) > 1 {
					body = strings.Join(lines[1:], "\n")
				}
			} else {
				// Multiple commits - format as a bulleted list of subjects in chronological order
				var sb strings.Builder
				for i := len(messages) - 1; i >= 0; i-- {
					msg := messages[i]
					// Get just the subject (first line)
					subject := strings.TrimSpace(strings.SplitN(msg, "\n", 2)[0])
					if subject != "" {
						sb.WriteString(subject + "\n")
					}
				}
				body = strings.TrimSpace(sb.String())
			}
		}
	}

	if !editInline {
		return body, nil
	}

	// Use editor for body editing
	return tui.OpenEditor(body, "stackit-pr-description-*.md")
}

// GetReviewers gets reviewers from flag or prompts user
func GetReviewers(reviewersFlag string, _ *runtime.Context) ([]string, []string, error) {
	if reviewersFlag == "" {
		// Don't prompt by default - return empty
		return nil, nil, nil
	}

	// Parse reviewers
	reviewers, teamReviewers := github.ParseReviewers(reviewersFlag)
	return reviewers, teamReviewers, nil
}

// GetReviewersWithPrompt gets reviewers, prompting if flag is empty
func GetReviewersWithPrompt(reviewersFlag string, _ *runtime.Context) ([]string, []string, error) {
	if reviewersFlag == "" {
		// Prompt for reviewers
		result, err := tui.PromptTextInput("Reviewers (comma-separated GitHub usernames):", "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get reviewers: %w", err)
		}

		reviewersFlag = result
	}

	// Parse reviewers
	reviewers, teamReviewers := github.ParseReviewers(reviewersFlag)
	return reviewers, teamReviewers, nil
}

// PreparePRMetadata prepares PR metadata for a branch
func PreparePRMetadata(branchName string, opts MetadataOptions, eng engine.Engine, ctx *runtime.Context) (*PRMetadata, error) {
	prInfo, _ := eng.GetPrInfo(ctx.Context, branchName)

	metadata := &PRMetadata{
		Title:   getStringValue(prInfo, "Title"),
		Body:    getStringValue(prInfo, "Body"),
		IsDraft: false,
	}

	// Determine if we should edit
	shouldEditTitle := opts.EditTitle || (opts.Edit && !opts.NoEditTitle)
	shouldEditBody := opts.EditDescription || (opts.Edit && !opts.NoEditDescription)

	// Use pre-generated AI metadata if provided
	aiTitle := opts.AIGeneratedTitle
	aiBody := opts.AIGeneratedBody

	// Try AI generation if enabled and no existing body/title and no pre-generated content
	if opts.AI && opts.AIClient != nil && aiTitle == "" && aiBody == "" && (metadata.Body == "" || (prInfo == nil || prInfo.Title == "")) {
		ctx.Splog.Debug("AI enabled, collecting PR context for branch %s", branchName)

		// Collect PR context
		prContext, err := ai.CollectPRContext(ctx, eng, branchName)
		if err != nil {
			ctx.Splog.Debug("Failed to collect PR context: %v, falling back to default", err)
		} else {
			// Generate PR description using AI
			generatedTitle, generatedBody, err := opts.AIClient.GeneratePRDescription(ctx.Context, prContext)
			if err != nil {
				ctx.Splog.Debug("AI generation failed: %v, falling back to default", err)
			} else {
				aiTitle = generatedTitle
				aiBody = generatedBody
				ctx.Splog.Debug("AI-generated PR description ready for review")
			}
		}
	}

	// Get title - use AI title if available and no existing title
	titleToUse := metadata.Title
	if aiTitle != "" && (prInfo == nil || prInfo.Title == "") {
		titleToUse = aiTitle
	}

	if shouldEditTitle || (prInfo == nil || prInfo.Title == "") {
		title, err := GetPRTitle(branchName, shouldEditTitle, titleToUse, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Title = title
	}

	// Get body - use AI body if available and no existing body
	bodyToUse := metadata.Body
	if aiBody != "" && bodyToUse == "" {
		bodyToUse = aiBody
	}

	if shouldEditBody || (prInfo == nil || prInfo.Body == "") {
		// Get body (with AI-generated content as initial value if available)
		finalBody, err := GetPRBody(branchName, shouldEditBody, bodyToUse, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Body = finalBody
	}

	// Get draft status - respect flags, default to published (not draft)
	switch {
	case opts.Draft:
		metadata.IsDraft = true
	case opts.Publish:
		metadata.IsDraft = false
	case prInfo == nil:
		// New PR - default to published (not draft)
		metadata.IsDraft = false
	default:
		metadata.IsDraft = prInfo.IsDraft
	}

	// Get reviewers
	if opts.ReviewersPrompt {
		reviewers, teamReviewers, err := GetReviewersWithPrompt(opts.Reviewers, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Reviewers = reviewers
		metadata.TeamReviewers = teamReviewers
	} else if opts.Reviewers != "" {
		reviewers, teamReviewers, err := GetReviewers(opts.Reviewers, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Reviewers = reviewers
		metadata.TeamReviewers = teamReviewers
	}

	// Save metadata to engine (in case command fails)
	if err := eng.UpsertPrInfo(ctx.Context, branchName, &engine.PrInfo{
		Title:   metadata.Title,
		Body:    metadata.Body,
		IsDraft: metadata.IsDraft,
	}); err != nil {
		ctx.Splog.Debug("Failed to save PR metadata: %v", err)
	}

	return metadata, nil
}

// MetadataOptions contains options for PR metadata collection
type MetadataOptions struct {
	Edit              bool
	EditTitle         bool
	EditDescription   bool
	NoEdit            bool
	NoEditTitle       bool
	NoEditDescription bool
	Draft             bool
	Publish           bool
	Reviewers         string
	ReviewersPrompt   bool
	AI                bool
	AIClient          ai.Client
	AIGeneratedTitle  string
	AIGeneratedBody   string
}

// PRMetadata contains PR metadata
type PRMetadata struct {
	Title         string
	Body          string
	IsDraft       bool
	Reviewers     []string
	TeamReviewers []string
}

// Helper to get string value from prInfo
func getStringValue(prInfo *engine.PrInfo, field string) string {
	if prInfo == nil {
		return ""
	}
	switch field {
	case "Title":
		return prInfo.Title
	case "Body":
		return prInfo.Body
	case "Base":
		return prInfo.Base
	case "State":
		return prInfo.State
	default:
		return ""
	}
}
