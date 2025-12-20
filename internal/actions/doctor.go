package actions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// DoctorOptions contains options for the doctor command
type DoctorOptions struct {
	Fix bool
}

// DoctorAction runs diagnostic checks on the stackit environment and repository
func DoctorAction(ctx *runtime.Context, opts DoctorOptions) error {
	splog := ctx.Splog
	eng := ctx.Engine

	if opts.Fix {
		splog.Info("Running stackit doctor with --fix...")
	} else {
		splog.Info("Running stackit doctor...")
	}
	splog.Newline()

	var warnings []string
	var errors []string

	// Environment checks
	splog.Info("Environment:")
	warnings, errors = checkEnvironment(splog, warnings, errors)

	splog.Newline()

	// Repository checks
	splog.Info("Repository:")
	warnings, errors = checkRepository(ctx, splog, warnings, errors)

	splog.Newline()

	// Stack state checks
	splog.Info("Stack State:")
	warnings, errors = checkStackState(eng, splog, warnings, errors, opts.Fix)

	// Summary
	splog.Newline()
	if len(errors) > 0 {
		splog.Warn("Doctor found %d error(s) and %d warning(s).", len(errors), len(warnings))
		for _, err := range errors {
			splog.Warn("  ❌ %s", err)
		}
		for _, warn := range warnings {
			splog.Warn("  ⚠️  %s", warn)
		}
		return fmt.Errorf("doctor found %d error(s)", len(errors))
	} else if len(warnings) > 0 {
		if opts.Fix {
			splog.Info("Doctor found %d warning(s), some of which may have been fixed.", len(warnings))
		} else {
			splog.Info("Doctor found %d warning(s). Your stackit setup is mostly healthy.", len(warnings))
		}
		for _, warn := range warnings {
			splog.Warn("  ⚠️  %s", warn)
		}
	} else {
		splog.Info("✅ All checks passed. Your stackit setup is healthy.")
	}

	return nil
}

// checkEnvironment performs environment-related checks
func checkEnvironment(splog *tui.Splog, warnings []string, errors []string) ([]string, []string) {
	// Check git version
	gitVersion, err := exec.Command("git", "version").Output()
	if err != nil {
		errors = append(errors, "git is not installed or not in PATH")
		splog.Warn("  ❌ git is not installed or not in PATH")
	} else {
		version := strings.TrimSpace(string(gitVersion))
		splog.Info("  ✅ %s", version)
	}

	// Check gh CLI
	ghVersion, err := exec.Command("gh", "version").Output()
	if err != nil {
		warnings = append(warnings, "GitHub CLI (gh) is not installed or not in PATH")
		splog.Warn("  ⚠️  GitHub CLI (gh) is not installed or not in PATH")
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
		splog.Warn("  ⚠️  GitHub authentication not configured")
	} else {
		if token == "" {
			warnings = append(warnings, "GitHub token is empty")
			splog.Warn("  ⚠️  GitHub token is empty")
		} else {
			// Try to create a GitHub client to verify connectivity
			ghCtx := context.Background()
			client, err := github.NewRealGitHubClient(ghCtx)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("GitHub authentication failed: %v", err))
				splog.Warn("  ⚠️  GitHub authentication failed: %v", err)
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

// checkRepository performs repository-related checks
func checkRepository(ctx *runtime.Context, splog *tui.Splog, warnings []string, errors []string) ([]string, []string) {
	// Check if we're in a git repository
	repoRoot := ctx.RepoRoot
	if repoRoot == "" {
		var err error
		if err = git.InitDefaultRepo(); err != nil {
			errors = append(errors, "not in a git repository")
			splog.Warn("  ❌ not in a git repository")
			return warnings, errors
		}
		repoRoot, err = git.GetRepoRoot()
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to get repo root: %v", err))
			splog.Warn("  ❌ failed to get repo root: %v", err)
			return warnings, errors
		}
	}
	splog.Info("  ✅ Current directory is a git repository")

	// Check remote configuration
	remoteURL, err := git.RunGitCommandWithContext(ctx.Context, "config", "--get", "remote.origin.url")
	if err != nil {
		warnings = append(warnings, "remote 'origin' is not configured")
		splog.Warn("  ⚠️  remote 'origin' is not configured")
	} else {
		// Check if it's a GitHub remote
		repoInfo, err := github.ParseGitHubRemoteURL(remoteURL)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("remote 'origin' is not a GitHub repository: %s", remoteURL))
			splog.Warn("  ⚠️  remote 'origin' is not a GitHub repository")
		} else {
			splog.Info("  ✅ Remote 'origin' is configured to GitHub (%s/%s)", repoInfo.Owner, repoInfo.Repo)
		}
	}

	// Check trunk branch
	trunk, err := config.GetTrunk(repoRoot)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get trunk: %v", err))
		splog.Warn("  ❌ failed to get trunk: %v", err)
	} else {
		// Check if trunk branch exists
		_, err = git.GetRevision(ctx.Context, trunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("trunk branch '%s' does not exist", trunk))
			splog.Warn("  ❌ trunk branch '%s' does not exist", trunk)
		} else {
			splog.Info("  ✅ Trunk branch '%s' exists", trunk)
		}
	}

	// Check if stackit is initialized
	if !config.IsInitialized(repoRoot) {
		errors = append(errors, "stackit is not initialized (run 'stackit init')")
		splog.Warn("  ❌ stackit is not initialized")
	} else {
		splog.Info("  ✅ stackit is initialized")
	}

	return warnings, errors
}

// checkStackState performs stack state and metadata integrity checks
func checkStackState(eng engine.Engine, splog *tui.Splog, warnings []string, errors []string, fix bool) ([]string, []string) {
	// Get all branches
	allBranches, err := git.GetAllBranchNames()
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get branch names: %v", err))
		splog.Warn("  ❌ failed to get branch names: %v", err)
		return warnings, errors
	}

	// Get all metadata refs
	metadataRefs, err := git.GetMetadataRefList()
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get metadata refs: %v", err))
		splog.Warn("  ❌ failed to get metadata refs: %v", err)
		return warnings, errors
	}

	// Check for orphaned metadata (metadata for branches that don't exist)
	branchSet := make(map[string]bool)
	for _, branch := range allBranches {
		branchSet[branch] = true
	}

	orphanedCount := 0
	prunedCount := 0
	for branchName := range metadataRefs {
		if !branchSet[branchName] {
			orphanedCount++
			if fix {
				if err := git.DeleteMetadataRef(branchName); err != nil {
					splog.Warn("  ❌ Failed to prune orphaned metadata for %s: %v", branchName, err)
					warnings = append(warnings, fmt.Sprintf("orphaned metadata found for deleted branch '%s' (fix failed)", branchName))
				} else {
					splog.Info("  ✅ Pruned orphaned metadata for deleted branch %s", tui.ColorBranchName(branchName, false))
					prunedCount++
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("orphaned metadata found for deleted branch '%s'", branchName))
			}
		}
	}

	if orphanedCount > 0 {
		if fix {
			if prunedCount == orphanedCount {
				splog.Info("  ✅ All %d orphaned metadata ref(s) pruned", prunedCount)
			} else {
				splog.Warn("  ⚠️  Found %d orphaned metadata ref(s), pruned %d", orphanedCount, prunedCount)
			}
		} else {
			splog.Warn("  ⚠️  Found %d orphaned metadata ref(s) (run 'stackit doctor --fix' to prune)", orphanedCount)
		}
	} else {
		splog.Info("  ✅ No orphaned metadata found")
	}

	// Check for corrupted metadata
	corruptedCount := 0
	for branchName := range metadataRefs {
		meta, err := git.ReadMetadataRef(branchName)
		if err != nil {
			corruptedCount++
			errors = append(errors, fmt.Sprintf("corrupted metadata for branch '%s': %v", branchName, err))
		} else if meta != nil {
			// Validate that if parent is set, it's not empty
			if meta.ParentBranchName != nil && *meta.ParentBranchName == "" {
				corruptedCount++
				errors = append(errors, fmt.Sprintf("invalid metadata for branch '%s': parent branch name is empty", branchName))
			}
		}
	}

	if corruptedCount > 0 {
		splog.Warn("  ❌ Found %d corrupted metadata ref(s)", corruptedCount)
	} else {
		splog.Info("  ✅ Metadata integrity check passed")
	}

	// Check for cycles in the stack graph
	cycles := detectCycles(eng)
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			errors = append(errors, fmt.Sprintf("cycle detected in stack graph: %s", strings.Join(cycle, " -> ")))
		}
		splog.Warn("  ❌ Found %d cycle(s) in stack graph", len(cycles))
	} else {
		splog.Info("  ✅ No cycles detected in stack graph")
	}

	// Check for missing parent branches
	missingParents := checkMissingParents(eng, allBranches)
	if len(missingParents) > 0 {
		for _, branch := range missingParents {
			parent := eng.GetParent(branch)
			warnings = append(warnings, fmt.Sprintf("branch '%s' has parent '%s' that does not exist", branch, parent))
		}
		splog.Warn("  ⚠️  Found %d branch(es) with missing parents", len(missingParents))
	} else {
		splog.Info("  ✅ All parent branches exist")
	}

	return warnings, errors
}

// detectCycles detects cycles in the branch parent graph using DFS
func detectCycles(eng engine.Engine) [][]string {
	var cycles [][]string
	allBranches := eng.AllBranchNames()
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parentMap := make(map[string]string)
	trunk := eng.Trunk()

	// Build parent map
	for _, branch := range allBranches {
		parent := eng.GetParent(branch)
		if parent != "" && parent != trunk {
			parentMap[branch] = parent
		}
	}

	var dfs func(string, []string)
	dfs = func(branch string, path []string) {
		if recStack[branch] {
			// Found a cycle - find where the cycle starts in the path
			cycleStart := -1
			for i, b := range path {
				if b == branch {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				// Extract the cycle: from first occurrence to current
				cycle := append(path[cycleStart:], branch)
				cycles = append(cycles, cycle)
			}
			return
		}

		if visited[branch] {
			// Already fully explored this branch
			return
		}

		visited[branch] = true
		recStack[branch] = true

		// Follow parent if it exists
		if parent, hasParent := parentMap[branch]; hasParent {
			dfs(parent, append(path, branch))
		}

		recStack[branch] = false
	}

	// Run DFS on all branches
	for _, branch := range allBranches {
		if branch != trunk && !visited[branch] {
			dfs(branch, []string{})
		}
	}

	return cycles
}

// checkMissingParents checks for branches whose parent branches don't exist
func checkMissingParents(eng engine.Engine, allBranches []string) []string {
	var missing []string
	branchSet := make(map[string]bool)
	trunk := eng.Trunk()

	for _, branch := range allBranches {
		branchSet[branch] = true
	}

	for _, branch := range allBranches {
		if branch == trunk {
			continue
		}
		parent := eng.GetParent(branch)
		if parent != "" && parent != trunk {
			if !branchSet[parent] {
				missing = append(missing, branch)
			}
		}
	}

	return missing
}

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
