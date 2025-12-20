package git

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

// RealGitHubClient implements GitHubClient using the real GitHub API
type RealGitHubClient struct {
	client *github.Client
	owner  string
	repo   string
}

// NewRealGitHubClient creates a new RealGitHubClient
func NewRealGitHubClient(ctx context.Context) (*RealGitHubClient, error) {
	token, err := getGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	repoInfo, err := getRepoInfoWithHostname(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	client, err := createGitHubClient(ctx, repoInfo.Hostname, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &RealGitHubClient{
		client: client,
		owner:  repoInfo.Owner,
		repo:   repoInfo.Repo,
	}, nil
}

// GetOwnerRepo returns the repository owner and name
func (c *RealGitHubClient) GetOwnerRepo() (string, string) {
	return c.owner, c.repo
}

// CreatePullRequest creates a new pull request
func (c *RealGitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePROptions) (*PullRequestInfo, error) {
	pr := &github.NewPullRequest{
		Title: github.String(opts.Title),
		Head:  github.String(opts.Head),
		Base:  github.String(opts.Base),
		Draft: github.Bool(opts.Draft),
	}

	if opts.Body != "" {
		pr.Body = github.String(opts.Body)
	}

	createdPR, _, err := c.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Add reviewers if specified
	if len(opts.Reviewers) > 0 || len(opts.TeamReviewers) > 0 {
		_, _, _ = c.client.PullRequests.RequestReviewers(ctx, owner, repo, *createdPR.Number, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
	}

	return toPullRequestInfo(createdPR), nil
}

// UpdatePullRequest updates an existing pull request
func (c *RealGitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts UpdatePROptions) error {
	// Handle draft status changes via GraphQL
	if opts.Draft != nil {
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err == nil && pr.Draft != nil {
			currentDraft := *pr.Draft
			desiredDraft := *opts.Draft

			if currentDraft != desiredDraft {
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

	_, _, err := c.client.PullRequests.Edit(ctx, owner, repo, prNumber, update)
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	// Update reviewers if specified
	if len(opts.Reviewers) > 0 || len(opts.TeamReviewers) > 0 {
		_, _, _ = c.client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
			Reviewers:     opts.Reviewers,
			TeamReviewers: opts.TeamReviewers,
		})
	}

	// Rerequest review if specified
	if opts.RerequestReview {
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
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
				_, _ = c.client.PullRequests.RemoveReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
					Reviewers:     reviewers,
					TeamReviewers: teamReviewers,
				})
				_, _, _ = c.client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{
					Reviewers:     reviewers,
					TeamReviewers: teamReviewers,
				})
			}
		}
	}

	return nil
}

// GetPullRequestByBranch gets a pull request for a branch
func (c *RealGitHubClient) GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*PullRequestInfo, error) {
	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
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

	return toPullRequestInfo(prs[0]), nil
}

// MergePullRequest merges a pull request
func (c *RealGitHubClient) MergePullRequest(ctx context.Context, branchName string) error {
	return MergePullRequest(ctx, c.client, c.owner, c.repo, branchName)
}

// GetPRChecksStatus returns the check status for a PR
func (c *RealGitHubClient) GetPRChecksStatus(ctx context.Context, branchName string) (bool, bool, error) {
	return GetPRChecksStatus(ctx, c.client, c.owner, c.repo, branchName)
}

// toPullRequestInfo converts a github.PullRequest to PullRequestInfo
func toPullRequestInfo(pr *github.PullRequest) *PullRequestInfo {
	if pr == nil {
		return nil
	}

	info := &PullRequestInfo{}

	if pr.Number != nil {
		info.Number = *pr.Number
	}
	if pr.NodeID != nil {
		info.NodeID = *pr.NodeID
	}
	if pr.HTMLURL != nil {
		info.HTMLURL = *pr.HTMLURL
	}
	if pr.Title != nil {
		info.Title = *pr.Title
	}
	if pr.Body != nil {
		info.Body = *pr.Body
	}
	if pr.State != nil {
		info.State = *pr.State
	}
	if pr.Draft != nil {
		info.Draft = *pr.Draft
	}
	if pr.Base != nil && pr.Base.Ref != nil {
		info.Base = *pr.Base.Ref
	}
	if pr.Head != nil && pr.Head.Ref != nil {
		info.Head = *pr.Head.Ref
	}

	return info
}
