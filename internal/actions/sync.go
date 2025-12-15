package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// SyncOptions are options for the sync command
type SyncOptions struct {
	All     bool
	Force   bool
	Restack bool
	Engine  engine.Engine
	Splog   *output.Splog
}

// SyncAction performs the sync operation
func SyncAction(opts SyncOptions) error {
	eng := opts.Engine
	splog := opts.Splog

	// Check for uncommitted changes
	if hasUncommittedChanges() {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them before syncing")
	}

	// Pull trunk
	splog.Info("Pulling %s from remote...", output.ColorBranchName(eng.Trunk(), false))
	pullResult, err := eng.PullTrunk()
	if err != nil {
		return fmt.Errorf("failed to pull trunk: %w", err)
	}

	switch pullResult {
	case engine.PullDone:
		rev, _ := eng.GetRevision(eng.Trunk())
		revShort := rev
		if len(rev) > 7 {
			revShort = rev[:7]
		}
		splog.Info("%s fast-forwarded to %s.",
			output.ColorBranchName(eng.Trunk(), true),
			output.ColorDim(revShort))
	case engine.PullUnneeded:
		splog.Info("%s is up to date.", output.ColorBranchName(eng.Trunk(), true))
	case engine.PullConflict:
		splog.Warn("%s could not be fast-forwarded.", output.ColorBranchName(eng.Trunk(), false))

		// Prompt to overwrite (or use force flag)
		shouldReset := opts.Force
		if !shouldReset {
			// For now, if not force and interactive, we'll skip
			// In a full implementation, we would prompt here
			splog.Info("Skipping trunk reset. Use --force to overwrite trunk with remote version.")
		}

		if shouldReset {
			if err := eng.ResetTrunkToRemote(); err != nil {
				return fmt.Errorf("failed to reset trunk: %w", err)
			}
			rev, _ := eng.GetRevision(eng.Trunk())
			revShort := rev
			if len(rev) > 7 {
				revShort = rev[:7]
			}
			splog.Info("%s set to %s.",
				output.ColorBranchName(eng.Trunk(), true),
				output.ColorDim(revShort))
		}
	}

	// Sync PR info
	allBranches := eng.AllBranchNames()
	repoOwner, repoName, _ := getRepoInfo()
	if repoOwner != "" && repoName != "" {
		if err := git.SyncPrInfo(allBranches, repoOwner, repoName); err != nil {
			// Non-fatal, continue
			splog.Debug("Failed to sync PR info: %v", err)
		}
	}

	// Clean branches (delete merged/closed)
	branchesToRestack := []string{}

	splog.Info("Checking if any branches have been merged/closed and can be deleted...")
	cleanResult, err := CleanBranches(CleanBranchesOptions{
		Force:  opts.Force,
		Engine: eng,
		Splog:  splog,
	})
	if err != nil {
		return fmt.Errorf("failed to clean branches: %w", err)
	}

	// Add branches with new parents to restack list
	for _, branchName := range cleanResult.BranchesWithNewParents {
		upstack := eng.GetRelativeStackUpstack(branchName)
		branchesToRestack = append(branchesToRestack, upstack...)
		branchesToRestack = append(branchesToRestack, branchName)
	}

	// Restack if requested
	if !opts.Restack {
		splog.Tip("Try the --restack flag to automatically restack the current stack.")
		return nil
	}

	// Add current branch stack to restack list
	currentBranch := eng.CurrentBranch()
	if currentBranch != "" && eng.IsBranchTracked(currentBranch) {
		// Get full stack (up to trunk)
		stack := eng.GetRelativeStack(currentBranch, engine.Scope{RecursiveParents: true})
		// Add current branch and its stack
		branchesToRestack = append(branchesToRestack, currentBranch)
		branchesToRestack = append(branchesToRestack, stack...)
	} else if currentBranch != "" && eng.IsTrunk(currentBranch) {
		// If on trunk, restack all branches
		allBranches = eng.AllBranchNames()
		for _, branchName := range allBranches {
			if !eng.IsTrunk(branchName) && eng.IsBranchTracked(branchName) {
				branchesToRestack = append(branchesToRestack, branchName)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueBranches := []string{}
	for _, branchName := range branchesToRestack {
		if !seen[branchName] {
			seen[branchName] = true
			uniqueBranches = append(uniqueBranches, branchName)
		}
	}

	// Restack branches
	if len(uniqueBranches) > 0 {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repo root: %w", err)
		}
		if err := RestackBranches(uniqueBranches, eng, splog, repoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}

// hasUncommittedChanges checks if there are uncommitted changes
func hasUncommittedChanges() bool {
	// Check git status
	output, err := git.RunGitCommand("status", "--porcelain")
	if err != nil {
		return false
	}
	return output != ""
}

// getRepoInfo gets repository owner and name from git remote
func getRepoInfo() (string, string, error) {
	// Get remote URL
	url, err := git.RunGitCommand("config", "--get", "remote.origin.url")
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
