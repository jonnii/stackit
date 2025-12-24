package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"stackit.dev/stackit/internal/git"
)

// getGitHubToken gets the GitHub token (similar to internal/github/pr_info.go)
func getGitHubToken() (string, error) {
	// Try environment variable first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	// Try gh CLI
	output, err := git.RunGHCommandWithContext(context.Background(), "auth", "token")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	token := strings.TrimSpace(output)
	if token == "" {
		return "", fmt.Errorf("empty GitHub token")
	}

	return token, nil
}
