package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

// checkEnvironment performs environment-related checks
func checkEnvironment(splog *tui.Splog, warnings []string, errors []string) ([]string, []string) {
	// Check git version
	gitVersion, err := exec.Command("git", "version").Output()
	if err != nil {
		errors = append(errors, "git is not installed or not in PATH")
		splog.Error("  git is not installed or not in PATH")
	} else {
		version := strings.TrimSpace(string(gitVersion))
		splog.Info("  ✅ %s", version)
	}

	// Check gh CLI
	ghVersion, err := exec.Command("gh", "version").Output()
	if err != nil {
		warnings = append(warnings, "GitHub CLI (gh) is not installed or not in PATH")
		splog.Warn("  GitHub CLI (gh) is not installed or not in PATH")
	} else {
		version := strings.TrimSpace(string(ghVersion))
		// Extract just the version number
		parts := strings.Fields(version)
		if len(parts) > 0 {
			splog.Info("  ✅ gh %s", parts[0])
		} else {
			splog.Info("  ✅ %s", version)
		}
	}

	// Check GitHub authentication
	token, err := getGitHubToken()
	if err != nil {
		warnings = append(warnings, "GitHub authentication not configured (GITHUB_TOKEN env var or gh auth token)")
		splog.Warn("  GitHub authentication not configured")
	} else {
		if token == "" {
			warnings = append(warnings, "GitHub token is empty")
			splog.Warn("  GitHub token is empty")
		} else {
			// Try to create a GitHub client to verify connectivity
			ghCtx := context.Background()
			client, err := github.NewRealGitHubClient(ghCtx)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("GitHub authentication failed: %v", err))
				splog.Warn("  GitHub authentication failed: %v", err)
			} else {
				owner, repo := client.GetOwnerRepo()
				if owner != "" && repo != "" {
					splog.Info("  ✅ GitHub authentication successful (%s/%s)", owner, repo)
				} else {
					splog.Info("  ✅ GitHub authentication successful")
				}
			}
		}
	}

	return warnings, errors
}
