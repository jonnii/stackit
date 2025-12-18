// Package git provides low-level Git operations, including repository access,
// branch operations, commit information, PR operations, and metadata management.
package git

import (
	"context"
	"fmt"
	"time"
)

// GetUserName returns the Git user's name from git config
func GetUserName(ctx context.Context) (string, error) {
	username, err := RunGitCommandWithContext(ctx, "config", "user.name")
	if err != nil {
		return "", fmt.Errorf("failed to get git user name: %w", err)
	}
	return username, nil
}

// GetCurrentDate returns the current date in YYMMDD format
func GetCurrentDate() string {
	now := time.Now()
	return now.Format("060102")
}
