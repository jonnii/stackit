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
		// If on trunk, restack all branches - use GetRelativeStack which returns
		// branches in topological order (parent before children) via DFS
		stack := eng.GetRelativeStack(currentBranch, engine.Scope{RecursiveChildren: true})
		branchesToRestack = append(branchesToRestack, stack...)
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

	// Sort branches topologically (parents before children) for correct restack order
	sortedBranches := sortBranchesTopologically(uniqueBranches, eng)

	// Restack branches
	if len(sortedBranches) > 0 {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repo root: %w", err)
		}
		if err := RestackBranches(sortedBranches, eng, splog, repoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	return nil
}

// sortBranchesTopologically sorts branches so parents come before children.
// This ensures correct restack order (bottom of stack first).
func sortBranchesTopologically(branches []string, eng engine.Engine) []string {
	if len(branches) == 0 {
		return branches
	}

	// Build a set for quick lookup
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	// Calculate depth for each branch (distance from trunk)
	depths := make(map[string]int)
	var getDepth func(branch string) int
	getDepth = func(branch string) int {
		if depth, ok := depths[branch]; ok {
			return depth
		}
		if eng.IsTrunk(branch) {
			depths[branch] = 0
			return 0
		}
		parent := eng.GetParent(branch)
		if parent == "" || eng.IsTrunk(parent) {
			depths[branch] = 1
			return 1
		}
		depths[branch] = getDepth(parent) + 1
		return depths[branch]
	}

	// Calculate depth for all branches
	for _, branch := range branches {
		getDepth(branch)
	}

	// Sort by depth (parents first, then children)
	result := make([]string, len(branches))
	copy(result, branches)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if depths[result[i]] > depths[result[j]] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
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
