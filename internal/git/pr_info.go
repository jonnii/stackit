package git

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// SyncPrInfo syncs PR information for branches from GitHub
func SyncPrInfo(ctx context.Context, branchNames []string, repoOwner, repoName string) error {
	// Get GitHub token
	token, err := getGitHubToken()
	if err != nil {
		// If no token, skip PR syncing (non-fatal)
		return nil
	}

	// Create GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get repository info if not provided
	if repoOwner == "" || repoName == "" {
		owner, name, err := getRepoInfo(ctx)
		if err != nil {
			return nil // Skip if can't determine repo
		}
		repoOwner = owner
		repoName = name
	}

	// Fetch PR info for each branch
	for _, branchName := range branchNames {
		prInfo, err := getPRInfoForBranch(ctx, client, repoOwner, repoName, branchName)
		if err != nil {
			// Continue with other branches
			continue
		}

		if prInfo != nil {
			// Update metadata
			meta, err := ReadMetadataRef(branchName)
			if err != nil {
				meta = &Meta{}
			}

			if meta.PrInfo == nil {
				meta.PrInfo = &PrInfo{}
			}

			// Update PR info
			if prInfo.Number != nil {
				meta.PrInfo.Number = prInfo.Number
			}
			if prInfo.Base != nil && prInfo.Base.Ref != nil {
				meta.PrInfo.Base = prInfo.Base.Ref
			}
			if prInfo.HTMLURL != nil {
				meta.PrInfo.URL = prInfo.HTMLURL
			}
			if prInfo.Title != nil {
				meta.PrInfo.Title = prInfo.Title
			}
			if prInfo.Body != nil {
				meta.PrInfo.Body = prInfo.Body
			}
			if prInfo.Draft != nil {
				meta.PrInfo.IsDraft = prInfo.Draft
			}
			if prInfo.State != nil {
				state := strings.ToUpper(*prInfo.State)
				meta.PrInfo.State = &state
			}

			// Write updated metadata
			_ = WriteMetadataRef(branchName, meta)
		}
	}

	return nil
}

// getPRInfoForBranch gets PR info for a branch
func getPRInfoForBranch(ctx context.Context, client *github.Client, owner, repo, branchName string) (*github.PullRequest, error) {
	// List PRs for this branch
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head:  fmt.Sprintf("%s:%s", owner, branchName),
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

	return prs[0], nil
}

// getGitHubToken gets GitHub token from environment or gh CLI
func getGitHubToken() (string, error) {
	// Try environment variable first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	// Try gh CLI
	output, err := RunGHCommandWithContext(context.Background(), "auth", "token")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	token := strings.TrimSpace(output)
	if token == "" {
		return "", fmt.Errorf("empty GitHub token")
	}

	return token, nil
}

// getRepoInfo gets repository owner and name from git remote
func getRepoInfo(ctx context.Context) (string, string, error) {
	// Get remote URL
	url, err := RunGitCommandWithContext(ctx, "config", "--get", "remote.origin.url")
	if err != nil {
		return "", "", err
	}

	// Parse URL (handles both https and ssh formats)
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid remote URL")
	}

	repoName := parts[len(parts)-1]
	var owner string
	if strings.Contains(url, "@") {
		// SSH format: git@github.com:owner/repo
		sshParts := strings.Split(url, ":")
		if len(sshParts) < 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL")
		}
		pathParts := strings.Split(sshParts[1], "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL")
		}
		owner = pathParts[0]
	} else {
		// HTTPS format: https://github.com/owner/repo
		owner = parts[len(parts)-2]
	}

	return owner, repoName, nil
}
