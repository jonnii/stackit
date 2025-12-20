package demo

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
)

// prCounter is used to generate unique PR numbers
var prCounter int32 = 100

func init() {
	// Register the demo GitHub client factory with runtime package
	runtime.DemoGitHubClientFactory = func() github.GitHubClient {
		return NewDemoGitHubClient()
	}
}

// DemoGitHubClient implements github.GitHubClient for demo mode
type DemoGitHubClient struct {
	owner string
	repo  string
	// prs stores PR info by branch name
	prs map[string]*github.PullRequestInfo
}

// NewDemoGitHubClient creates a new demo GitHub client
func NewDemoGitHubClient() *DemoGitHubClient {
	return &DemoGitHubClient{
		owner: "example",
		repo:  "repo",
		prs:   make(map[string]*github.PullRequestInfo),
	}
}

// GetOwnerRepo returns the repository owner and name
func (c *DemoGitHubClient) GetOwnerRepo() (string, string) {
	return c.owner, c.repo
}

// CreatePullRequest creates a simulated pull request
func (c *DemoGitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, opts github.CreatePROptions) (*github.PullRequestInfo, error) {
	simulateDelay(delayMedium)

	prNum := int(atomic.AddInt32(&prCounter, 1))
	pr := &github.PullRequestInfo{
		Number:  prNum,
		NodeID:  fmt.Sprintf("PR_%d", prNum),
		HTMLURL: fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, prNum),
		Title:   opts.Title,
		Body:    opts.Body,
		State:   "open",
		Draft:   opts.Draft,
		Base:    opts.Base,
		Head:    opts.Head,
	}

	c.prs[opts.Head] = pr
	return pr, nil
}

// UpdatePullRequest simulates updating a pull request
func (c *DemoGitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts github.UpdatePROptions) error {
	simulateDelay(delayShort)

	// Find the PR by number
	for _, pr := range c.prs {
		if pr.Number == prNumber {
			if opts.Title != nil {
				pr.Title = *opts.Title
			}
			if opts.Body != nil {
				pr.Body = *opts.Body
			}
			if opts.Base != nil {
				pr.Base = *opts.Base
			}
			if opts.Draft != nil {
				pr.Draft = *opts.Draft
			}
			return nil
		}
	}

	return nil
}

// GetPullRequestByBranch returns a simulated PR for a branch
func (c *DemoGitHubClient) GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*github.PullRequestInfo, error) {
	simulateDelay(delayShort)

	if pr, ok := c.prs[branchName]; ok {
		return pr, nil
	}
	return nil, nil
}

// MergePullRequest simulates merging a pull request
func (c *DemoGitHubClient) MergePullRequest(ctx context.Context, branchName string) error {
	simulateDelay(delayMedium)

	if pr, ok := c.prs[branchName]; ok {
		pr.State = "closed"
	}
	return nil
}

// GetPRChecksStatus returns simulated check status
func (c *DemoGitHubClient) GetPRChecksStatus(ctx context.Context, branchName string) (bool, bool, error) {
	// Simulate a small delay
	time.Sleep(50 * time.Millisecond)

	// In demo mode, always return checks passing
	return true, false, nil
}
