package testhelpers

import (
	"context"

	"github.com/google/go-github/v62/github"

	"stackit.dev/stackit/internal/git"
)

// MockGitHubClient implements git.GitHubClient using the mock server
type MockGitHubClient struct {
	client *github.Client
	owner  string
	repo   string
	config *MockGitHubServerConfig
}

// NewMockGitHubClientInterface creates a GitHubClient interface implementation
// using the mock server
func NewMockGitHubClientInterface(client *github.Client, owner, repo string, config *MockGitHubServerConfig) git.GitHubClient {
	return &MockGitHubClient{
		client: client,
		owner:  owner,
		repo:   repo,
		config: config,
	}
}

// GetOwnerRepo returns the repository owner and name
func (c *MockGitHubClient) GetOwnerRepo() (string, string) {
	return c.owner, c.repo
}

// CreatePullRequest creates a new pull request
func (c *MockGitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, opts git.CreatePROptions) (*git.PullRequestInfo, error) {
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
		return nil, err
	}

	return toPullRequestInfo(createdPR), nil
}

// UpdatePullRequest updates an existing pull request
func (c *MockGitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts git.UpdatePROptions) error {
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
	return err
}

// GetPullRequestByBranch gets a pull request for a branch
func (c *MockGitHubClient) GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*git.PullRequestInfo, error) {
	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head:  owner + ":" + branchName,
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(prs) == 0 {
		return nil, nil
	}

	return toPullRequestInfo(prs[0]), nil
}

// MergePullRequest merges a pull request
func (c *MockGitHubClient) MergePullRequest(ctx context.Context, branchName string) error {
	// In tests, just return nil
	return nil
}

// GetPRChecksStatus returns the check status for a PR
func (c *MockGitHubClient) GetPRChecksStatus(ctx context.Context, branchName string) (bool, bool, error) {
	// In tests, always return passing
	return true, false, nil
}

// toPullRequestInfo converts a github.PullRequest to git.PullRequestInfo
func toPullRequestInfo(pr *github.PullRequest) *git.PullRequestInfo {
	if pr == nil {
		return nil
	}

	info := &git.PullRequestInfo{}

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
