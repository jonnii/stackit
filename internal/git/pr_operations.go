package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
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
		_, _, err = client.PullRequests.RequestReviewers(ctx, owner, repo, *createdPR.Number, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
		if err != nil {
			// Non-fatal, log but continue
			// We could return an error here, but for now we'll just continue
		}
	}

	return createdPR, nil
}

// UpdatePullRequest updates an existing pull request
func UpdatePullRequest(ctx context.Context, client *github.Client, owner, repo string, prNumber int, opts UpdatePROptions) error {
	// Handle draft status changes separately using GitHub CLI, as the REST API
	// doesn't support updating draft status. We need to use GraphQL mutation
	// markPullRequestReadyForReview, which is easier via gh CLI.
	if opts.Draft != nil {
		// Get current PR to check if draft status actually needs to change
		pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err == nil && pr.Draft != nil {
			currentDraft := *pr.Draft
			desiredDraft := *opts.Draft

			// Only change draft status if it's different
			if currentDraft != desiredDraft {
				if desiredDraft {
					// Mark as draft (convert ready PR to draft)
					cmd := exec.Command("gh", "pr", "ready", fmt.Sprintf("%d", prNumber), "--undo")
					if output, err := cmd.CombinedOutput(); err != nil {
						return fmt.Errorf("failed to mark PR %d as draft: %w (output: %s)", prNumber, err, string(output))
					}
				} else {
					// Mark as ready for review (publish draft PR)
					cmd := exec.Command("gh", "pr", "ready", fmt.Sprintf("%d", prNumber))
					if output, err := cmd.CombinedOutput(); err != nil {
						return fmt.Errorf("failed to mark PR %d as ready for review: %w (output: %s)", prNumber, err, string(output))
					}
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
		_, _, err = client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
		if err != nil {
			// Non-fatal, log but continue
		}
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
				_, err = client.PullRequests.RemoveReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
					Reviewers:     reviewers,
					TeamReviewers: teamReviewers,
				})
				if err == nil {
					_, _, err = client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
						Reviewers:     reviewers,
						TeamReviewers: teamReviewers,
					})
				}
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

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get repository info
	owner, repo, err := getRepoInfo()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get repository info: %w", err)
	}

	return client, owner, repo, nil
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

// MergePullRequest merges a pull request using GitHub CLI
func MergePullRequest(branchName string) error {
	// Use gh CLI to merge the PR
	cmd := exec.Command("gh", "pr", "merge", branchName, "--merge")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to merge PR for branch %s: %w (output: %s)", branchName, err, string(output))
	}
	return nil
}
