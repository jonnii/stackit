// Package github provides a client for interacting with the GitHub API.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

const (
	// GitHub check conclusion and status constants
	checkConclusionFailure        = "FAILURE"
	checkConclusionCanceled       = "CANCELED"
	checkConclusionTimedOut       = "TIMED_OUT"
	checkConclusionActionRequired = "ACTION_REQUIRED"
	checkStateFailure             = "FAILURE"
	checkStateError               = "ERROR"
	checkStatePending             = "PENDING"
)

// CreatePROptions contains options for creating a pull request
type CreatePROptions struct {
	Title         string
	Body          string
	Head          string
	Base          string
	Draft         bool
	Reviewers     []string
	TeamReviewers []string
}

// UpdatePROptions contains options for updating a pull request
type UpdatePROptions struct {
	Title           *string
	Body            *string
	Base            *string
	Draft           *bool
	Reviewers       []string
	TeamReviewers   []string
	MergeWhenReady  *bool
	RerequestReview bool
}

// CreatePullRequest creates a new pull request
func CreatePullRequest(ctx context.Context, client *github.Client, owner, repo string, opts CreatePROptions) (*github.PullRequest, error) {
	pr := &github.NewPullRequest{
		Title: github.String(opts.Title),
		Head:  github.String(opts.Head),
		Base:  github.String(opts.Base),
		Draft: github.Bool(opts.Draft),
	}

	if opts.Body != "" {
		pr.Body = github.String(opts.Body)
	}

	createdPR, _, err := client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Add reviewers if specified
	if len(opts.Reviewers) > 0 || len(opts.TeamReviewers) > 0 {
		_, _, _ = client.PullRequests.RequestReviewers(ctx, owner, repo, *createdPR.Number, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
	}

	return createdPR, nil
}

// UpdatePullRequest updates an existing pull request
func UpdatePullRequest(ctx context.Context, client *github.Client, owner, repo string, prNumber int, opts UpdatePROptions) error {
	// Handle draft status changes separately using GraphQL API, as the REST API
	// doesn't support updating draft status. We need to use GraphQL mutation
	// markPullRequestReadyForReview or convertPullRequestToDraft.
	if opts.Draft != nil {
		// Get current PR to check if draft status actually needs to change
		pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err == nil && pr.Draft != nil {
			currentDraft := *pr.Draft
			desiredDraft := *opts.Draft

			// Only change draft status if it's different
			if currentDraft != desiredDraft {
				// Get the PR's Node ID (required for GraphQL)
				if pr.NodeID == nil {
					return fmt.Errorf("PR %d does not have a Node ID", prNumber)
				}

				if err := updatePRDraftStatus(ctx, *pr.NodeID, desiredDraft); err != nil {
					return fmt.Errorf("failed to update draft status for PR %d: %w", prNumber, err)
				}
			}
		}
	}

	// Update other fields via REST API
	update := &github.PullRequest{}

	if opts.Title != nil {
		update.Title = opts.Title
	}
	if opts.Body != nil {
		update.Body = opts.Body
	}
	if opts.Base != nil {
		update.Base = &github.PullRequestBranch{
			Ref: opts.Base,
		}
	}
	// Note: We don't set update.Draft here because the REST API doesn't support it

	_, _, err := client.PullRequests.Edit(ctx, owner, repo, prNumber, update)
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	// Update reviewers if specified
	if len(opts.Reviewers) > 0 || len(opts.TeamReviewers) > 0 {
		_, _, _ = client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
	}

	// Rerequest review if specified
	if opts.RerequestReview {
		// Get current reviewers first
		pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err == nil && pr.RequestedReviewers != nil {
			var reviewers []string
			var teamReviewers []string
			for _, reviewer := range pr.RequestedReviewers {
				reviewers = append(reviewers, *reviewer.Login)
			}
			for _, team := range pr.RequestedTeams {
				teamReviewers = append(teamReviewers, *team.Slug)
			}
			if len(reviewers) > 0 || len(teamReviewers) > 0 {
				// Remove and re-add reviewers
				_, _ = client.PullRequests.RemoveReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
					Reviewers:     reviewers,
					TeamReviewers: teamReviewers,
				})
				_, _, _ = client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
					Reviewers:     reviewers,
					TeamReviewers: teamReviewers,
				})
			}
		}
	}

	// Merge when ready (this is typically handled via GitHub's auto-merge feature)
	// For now, we'll skip this as it requires additional API calls and permissions

	return nil
}

// GetPullRequestByBranch gets a pull request for a branch
func GetPullRequestByBranch(ctx context.Context, client *github.Client, owner, repo, branchName string) (*github.PullRequest, error) {
	// List PRs for this branch
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head:  fmt.Sprintf("%s:%s", owner, branchName),
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil
	}

	return prs[0], nil
}

// GetGitHubClient creates a GitHub client with authentication
func GetGitHubClient(ctx context.Context) (*github.Client, string, string, error) {
	token, err := getGitHubToken()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	repoInfo, err := getRepoInfoWithHostname(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get repository info: %w", err)
	}

	client, err := createGitHubClient(ctx, repoInfo.Hostname, token)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return client, repoInfo.Owner, repoInfo.Repo, nil
}

// ParseReviewers parses a comma-separated string of reviewers
// Returns individual reviewers and team reviewers
// Team reviewers can be specified as "org/team" or just "team"
func ParseReviewers(reviewersStr string) ([]string, []string) {
	if reviewersStr == "" {
		return nil, nil
	}

	var reviewers []string
	var teamReviewers []string

	parts := strings.Split(reviewersStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a team (contains /)
		if strings.Contains(part, "/") {
			// Could be org/team or just a team slug
			teamReviewers = append(teamReviewers, part)
		} else {
			reviewers = append(reviewers, part)
		}
	}

	return reviewers, teamReviewers
}

// MergePullRequest merges a pull request using the GitHub API
func MergePullRequest(ctx context.Context, client *github.Client, owner, repo, branchName string) error {
	// First, get the PR for this branch
	pr, err := GetPullRequestByBranch(ctx, client, owner, repo, branchName)
	if err != nil {
		return fmt.Errorf("failed to get PR for branch %s: %w", branchName, err)
	}
	if pr == nil {
		return fmt.Errorf("no PR found for branch %s", branchName)
	}

	// Merge the PR using merge method
	mergeRequest := &github.PullRequestOptions{
		MergeMethod: "merge",
	}
	_, _, err = client.PullRequests.Merge(ctx, owner, repo, *pr.Number, "", mergeRequest)
	if err != nil {
		return fmt.Errorf("failed to merge PR #%d for branch %s: %w", *pr.Number, branchName, err)
	}
	return nil
}

// GetPRChecksStatus returns the check status for a PR
func GetPRChecksStatus(ctx context.Context, client *github.Client, owner, repo, branchName string) (*CheckStatus, error) {
	// First, get the PR for this branch to get the head SHA
	pr, err := GetPullRequestByBranch(ctx, client, owner, repo, branchName)
	if err != nil {
		return &CheckStatus{Passing: true, Pending: false}, nil //nolint:nilerr
	}
	if pr == nil || pr.Head == nil || pr.Head.SHA == nil {
		return &CheckStatus{Passing: true, Pending: false}, nil
	}

	headSHA := *pr.Head.SHA

	// Get check runs for the head commit
	checkRuns, _, err := client.Checks.ListCheckRunsForRef(ctx, owner, repo, headSHA, &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})

	// Use a map to deduplicate checks by name, preferring check runs over status checks
	checkMap := make(map[string]CheckDetail)
	hasPending := false
	hasFailing := false

	// First, get check runs (more detailed information)
	if err == nil && checkRuns != nil {
		for _, run := range checkRuns.CheckRuns {
			detail := CheckDetail{
				Name:   run.GetName(),
				Status: strings.ToUpper(run.GetStatus()),
			}
			if run.Conclusion != nil {
				detail.Conclusion = strings.ToUpper(*run.Conclusion)
			}
			if run.StartedAt != nil {
				detail.StartedAt = run.StartedAt.Time
			}
			if run.CompletedAt != nil {
				detail.FinishedAt = run.CompletedAt.Time
			}
			checkMap[detail.Name] = detail

			if detail.Status == "QUEUED" || detail.Status == "IN_PROGRESS" {
				hasPending = true
			}
			if detail.Conclusion == checkConclusionFailure || detail.Conclusion == checkConclusionCanceled || detail.Conclusion == checkConclusionTimedOut || detail.Conclusion == checkConclusionActionRequired {
				hasFailing = true
			}
		}
	}

	// Also get combined status, but only add if we don't already have this check from check runs
	combinedStatus, _, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, headSHA, nil)
	if err == nil && combinedStatus != nil {
		for _, status := range combinedStatus.Statuses {
			name := status.GetContext()
			// Skip if we already have this check from check runs
			if _, exists := checkMap[name]; exists {
				continue
			}

			detail := CheckDetail{
				Name:   name,
				Status: "COMPLETED",
			}
			state := strings.ToUpper(status.GetState())
			switch state {
			case checkStatePending:
				detail.Status = "IN_PROGRESS"
				hasPending = true
			case checkStateFailure, checkStateError:
				detail.Conclusion = checkConclusionFailure
				hasFailing = true
			case "SUCCESS":
				detail.Conclusion = "SUCCESS"
			}
			// Combined status doesn't give us precise times usually in this struct
			checkMap[name] = detail
		}

		// If no checks at all but combined status shows something, use it for overall status
		if len(checkMap) == 0 && combinedStatus.State != nil {
			state := strings.ToUpper(*combinedStatus.State)
			if state == checkStatePending {
				hasPending = true
			} else if state == checkStateFailure || state == checkStateError {
				hasFailing = true
			}
		}
	}

	// Convert map to slice
	checks := make([]CheckDetail, 0, len(checkMap))
	for _, check := range checkMap {
		checks = append(checks, check)
	}

	return &CheckStatus{
		Passing: !hasFailing,
		Pending: hasPending,
		Checks:  checks,
	}, nil
}

// updatePRDraftStatus updates the draft status of a PR using GitHub's GraphQL API
func updatePRDraftStatus(ctx context.Context, pullRequestID string, isDraft bool) error {
	// Get GitHub token
	token, err := getGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Get repository info to determine hostname
	repoInfo, err := getRepoInfoWithHostname(ctx)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	// Construct GraphQL endpoint URL
	var graphqlURL string
	if repoInfo.Hostname == "github.com" {
		graphqlURL = "https://api.github.com/graphql"
	} else {
		// GitHub Enterprise: https://hostname/api/graphql
		graphqlURL = fmt.Sprintf("https://%s/api/graphql", repoInfo.Hostname)
	}

	// Create authenticated HTTP client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, ts)

	// Determine which mutation to use
	var mutation string
	var mutationName string
	if isDraft {
		mutationName = "convertPullRequestToDraft"
		mutation = `mutation ConvertPullRequestToDraft($pullRequestId: ID!) {
			convertPullRequestToDraft(input: {pullRequestId: $pullRequestId}) {
				pullRequest {
					id
					isDraft
				}
			}
		}`
	} else {
		mutationName = "markPullRequestReadyForReview"
		mutation = `mutation MarkPullRequestReadyForReview($pullRequestId: ID!) {
			markPullRequestReadyForReview(input: {pullRequestId: $pullRequestId}) {
				pullRequest {
					id
					isDraft
				}
			}
		}`
	}

	// Prepare GraphQL request
	requestBody := map[string]interface{}{
		"query": mutation,
		"variables": map[string]interface{}{
			"pullRequestId": pullRequestID,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	// Make GraphQL request
	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GraphQL response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check for GraphQL errors
	var graphqlResponse struct {
		Data   interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	if len(graphqlResponse.Errors) > 0 {
		errorMessages := make([]string, len(graphqlResponse.Errors))
		for i, err := range graphqlResponse.Errors {
			errorMessages[i] = err.Message
		}
		return fmt.Errorf("GraphQL %s mutation failed: %s", mutationName, strings.Join(errorMessages, "; "))
	}

	return nil
}
