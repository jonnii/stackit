package doctor

import (
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// checkRepository performs repository-related checks
func checkRepository(ctx *runtime.Context, splog *tui.Splog, warnings []string, errors []string, trunk string) ([]string, []string) {
	// Check if we're in a git repository
	if ctx.RepoRoot == "" {
		if err := git.InitDefaultRepo(); err != nil {
			errors = append(errors, "not in a git repository")
			splog.Error("  not in a git repository")
			return warnings, errors
		}
		if _, err := git.GetRepoRoot(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to get repo root: %v", err))
			splog.Error("  failed to get repo root: %v", err)
			return warnings, errors
		}
	}
	splog.Info("  ✅ Current directory is a git repository")

	// Check remote configuration
	remoteURL, err := git.RunGitCommandWithContext(ctx.Context, "config", "--get", "remote.origin.url")
	if err != nil {
		warnings = append(warnings, "remote 'origin' is not configured")
		splog.Warn("  remote 'origin' is not configured")
	} else {
		// Check if it's a GitHub remote
		repoInfo, err := github.ParseGitHubRemoteURL(remoteURL)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("remote 'origin' is not a GitHub repository: %s", remoteURL))
			splog.Warn("  remote 'origin' is not a GitHub repository")
		} else {
			splog.Info("  ✅ Remote 'origin' is configured to GitHub (%s/%s)", repoInfo.Owner, repoInfo.Repo)
		}
	}

	// Check trunk branch
	if trunk == "" {
		errors = append(errors, "trunk branch not configured")
		splog.Error("  trunk branch not configured")
	} else {
		// Check if trunk branch exists
		_, err := git.GetRevision(trunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("trunk branch '%s' does not exist", trunk))
			splog.Error("  trunk branch '%s' does not exist", trunk)
		} else {
			splog.Info("  ✅ Trunk branch '%s' exists", trunk)
		}
	}

	// Check if stackit is initialized (if trunk is set, it's initialized)
	if trunk == "" {
		errors = append(errors, "stackit is not initialized (run 'stackit init')")
		splog.Error("  stackit is not initialized")
	} else {
		splog.Info("  ✅ stackit is initialized")
	}

	return warnings, errors
}
