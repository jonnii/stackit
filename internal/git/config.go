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

// GetCurrentDate returns the current date and time in yyyyMMddHHmmss format in UTC
func GetCurrentDate() string {
	now := time.Now().UTC()
	return now.Format("20060102150405")
}
