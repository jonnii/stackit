package actions

import (
	"context"
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/git"
)

// StagingOptions defines which changes to stage
type StagingOptions struct {
	All    bool
	Update bool
	Patch  bool
}

// StageChanges stages changes based on the provided options
func StageChanges(ctx context.Context, opts StagingOptions) error {
	if opts.Patch && !opts.All {
		return git.StagePatch(ctx)
	}

	if opts.All {
		return git.StageAll(ctx)
	}

	if opts.Update {
		return git.StageTracked(ctx)
	}

	return nil
}

// GetRepoInfo gets repository owner and name from git remote
func GetRepoInfo(ctx context.Context) (string, string, error) {
	// Get remote URL
	url, err := git.RunGitCommandWithContext(ctx, "config", "--get", "remote.origin.url")
	if err != nil {
		return "", "", err
	}

	// Parse URL (handles both https and ssh formats)
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
