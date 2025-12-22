// Package github provides a client for interacting with the GitHub API.
package github

import (
	"context"
	"time"
)

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

// CheckDetail represents the status of an individual CI check
type CheckDetail struct {
	Name       string
	Status     string // QUEUED, IN_PROGRESS, COMPLETED
	Conclusion string // SUCCESS, FAILURE, NEUTRAL, etc.
	StartedAt  time.Time
	FinishedAt time.Time
}

// CheckStatus represents the combined status of all CI checks for a PR
type CheckStatus struct {
	Passing bool
	Pending bool
	Checks  []CheckDetail
}

// Client is an interface for GitHub API interactions
type Client interface {
	// CreatePullRequest creates a new pull request
	CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePROptions) (*PullRequestInfo, error)

	// UpdatePullRequest updates an existing pull request
	UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts UpdatePROptions) error

	// GetPullRequestByBranch gets a pull request for a branch
	GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*PullRequestInfo, error)

	// MergePullRequest merges a pull request
	MergePullRequest(ctx context.Context, branchName string) error

	// GetPRChecksStatus returns the check status for a PR
	GetPRChecksStatus(ctx context.Context, branchName string) (*CheckStatus, error)

	// GetOwnerRepo returns the repository owner and name
	GetOwnerRepo() (owner, repo string)
}
