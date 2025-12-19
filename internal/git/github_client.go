package git

import "context"

// PullRequestInfo contains information about a pull request
// This is a simplified struct to avoid coupling to go-github library
type PullRequestInfo struct {
	Number  int
	NodeID  string
	HTMLURL string
	Title   string
	Body    string
	State   string
	Draft   bool
	Base    string
	Head    string
}

// GitHubClient is an interface for GitHub API interactions
type GitHubClient interface {
	// CreatePullRequest creates a new pull request
	CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePROptions) (*PullRequestInfo, error)

	// UpdatePullRequest updates an existing pull request
	UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts UpdatePROptions) error

	// GetPullRequestByBranch gets a pull request for a branch
	GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*PullRequestInfo, error)

	// MergePullRequest merges a pull request
	MergePullRequest(ctx context.Context, branchName string) error

	// GetPRChecksStatus returns the check status for a PR
	// Returns (passing, pending, error)
	GetPRChecksStatus(ctx context.Context, branchName string) (passing bool, pending bool, err error)

	// GetOwnerRepo returns the repository owner and name
	GetOwnerRepo() (owner, repo string)
}
