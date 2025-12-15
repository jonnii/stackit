package actions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	stackitcontext "stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// GetPRTitle gets the PR title, prompting if needed
func GetPRTitle(branchName string, editInline bool, existingTitle string, ctx *stackitcontext.Context) (string, error) {
	// First check if we have a saved title
	title := existingTitle
	if title == "" {
		// Otherwise, use the subject of the oldest commit on the branch
		subject, err := git.GetCommitSubject(branchName)
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
	result, err := promptTextInput("Title:", title)
	if err != nil {
		return "", fmt.Errorf("failed to get PR title: %w", err)
	}

	return result, nil
}

// GetPRBody gets the PR body, prompting if needed
func GetPRBody(branchName string, editInline bool, existingBody string, ctx *stackitcontext.Context) (string, error) {
	body := existingBody
	if body == "" {
		// Infer from commit messages
		messages, err := git.GetCommitMessages(branchName)
		if err == nil && len(messages) > 0 {
			if len(messages) == 1 {
				// Single commit - use body (skip first line which is subject)
				lines := strings.Split(messages[0], "\n")
				if len(lines) > 1 {
					body = strings.Join(lines[1:], "\n")
				}
			} else {
				// Multiple commits - join all messages
				body = strings.Join(messages, "\n\n")
			}
		}
	}

	if !editInline {
		return body, nil
	}

	// Use editor for body editing
	return editPRBodyInEditor(body, ctx)
}

// editPRBodyInEditor opens an editor to edit the PR body
func editPRBodyInEditor(initialBody string, ctx *stackitcontext.Context) (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "stackit-pr-description-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial body
	if _, err := tmpFile.WriteString(initialBody); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// GetReviewers gets reviewers from flag or prompts user
func GetReviewers(reviewersFlag string, ctx *stackitcontext.Context) ([]string, []string, error) {
	if reviewersFlag == "" {
		// Don't prompt by default - return empty
		return nil, nil, nil
	}

	// Parse reviewers
	reviewers, teamReviewers := git.ParseReviewers(reviewersFlag)
	return reviewers, teamReviewers, nil
}

// GetReviewersWithPrompt gets reviewers, prompting if flag is empty
func GetReviewersWithPrompt(reviewersFlag string, ctx *stackitcontext.Context) ([]string, []string, error) {
	if reviewersFlag == "" {
		// Prompt for reviewers
		result, err := promptTextInput("Reviewers (comma-separated GitHub usernames):", "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get reviewers: %w", err)
		}

		reviewersFlag = result
	}

	// Parse reviewers
	reviewers, teamReviewers := git.ParseReviewers(reviewersFlag)
	return reviewers, teamReviewers, nil
}

// GetPRDraftStatus prompts user for draft status
func GetPRDraftStatus(ctx *stackitcontext.Context) (bool, error) {
	result, err := promptConfirm("Create as draft?", true)
	if err != nil {
		return false, fmt.Errorf("failed to get draft status: %w", err)
	}

	return result, nil
}

// PreparePRMetadata prepares PR metadata for a branch
func PreparePRMetadata(branchName string, opts SubmitMetadataOptions, eng engine.Engine, ctx *stackitcontext.Context) (*PRMetadata, error) {
	prInfo, _ := eng.GetPrInfo(branchName)

	metadata := &PRMetadata{
		Title:   getStringValue(prInfo, "Title"),
		Body:    getStringValue(prInfo, "Body"),
		IsDraft: false,
	}

	// Determine if we should edit
	shouldEditTitle := opts.EditTitle || (opts.Edit && !opts.NoEditTitle)
	shouldEditBody := opts.EditDescription || (opts.Edit && !opts.NoEditDescription)

	// Get title
	if shouldEditTitle || (prInfo == nil || prInfo.Title == "") {
		title, err := GetPRTitle(branchName, shouldEditTitle, metadata.Title, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Title = title
	}

	// Get body
	if shouldEditBody || (prInfo == nil || prInfo.Body == "") {
		body, err := GetPRBody(branchName, shouldEditBody, metadata.Body, ctx)
		if err != nil {
			return nil, err
		}
		metadata.Body = body
	}

	// Get draft status
	if opts.Draft {
		metadata.IsDraft = true
	} else if opts.Publish {
		metadata.IsDraft = false
	} else if prInfo == nil {
		// New PR - default to draft unless publish is set
		metadata.IsDraft = true
	} else {
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
	if err := eng.UpsertPrInfo(branchName, &engine.PrInfo{
		Title:   metadata.Title,
		Body:    metadata.Body,
		IsDraft: metadata.IsDraft,
	}); err != nil {
		ctx.Splog.Debug("Failed to save PR metadata: %v", err)
	}

	return metadata, nil
}

// SubmitMetadataOptions contains options for PR metadata collection
type SubmitMetadataOptions struct {
	Edit             bool
	EditTitle        bool
	EditDescription  bool
	NoEdit           bool
	NoEditTitle      bool
	NoEditDescription bool
	Draft            bool
	Publish          bool
	Reviewers        string
	ReviewersPrompt  bool
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

