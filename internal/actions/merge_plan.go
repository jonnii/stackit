package actions

import (
	"fmt"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// MergeStrategy defines how PRs in the stack should be merged
type MergeStrategy string

const (
	MergeStrategyBottomUp MergeStrategy = "bottom-up"
	MergeStrategyTopDown  MergeStrategy = "top-down"
)

// StepType represents the type of step in a merge plan
type StepType string

const (
	StepMergePR      StepType = "MERGE_PR"
	StepRestack      StepType = "RESTACK"
	StepDeleteBranch StepType = "DELETE_BRANCH"
	StepUpdatePRBase StepType = "UPDATE_PR_BASE"
	StepPullTrunk    StepType = "PULL_TRUNK"
)

// ChecksStatus represents the CI check status for a PR
type ChecksStatus string

const (
	ChecksPassing ChecksStatus = "PASSING"
	ChecksFailing ChecksStatus = "FAILING"
	ChecksPending ChecksStatus = "PENDING"
	ChecksNone    ChecksStatus = "NONE"
)

// BranchMergeInfo contains info about a branch to be merged
type BranchMergeInfo struct {
	BranchName    string
	PRNumber      int
	PRURL         string
	IsDraft       bool
	ChecksStatus  ChecksStatus
	MatchesRemote bool
}

// MergePlanStep represents a single step in the merge plan
type MergePlanStep struct {
	StepType    StepType
	BranchName  string
	PRNumber    int
	Description string // Human-readable description for display
}

// MergePlan is the complete plan for a merge operation
type MergePlan struct {
	Strategy        MergeStrategy
	CurrentBranch   string
	BranchesToMerge []BranchMergeInfo // Branches that will be merged (bottom to top)
	UpstackBranches []string          // Branches above current that will be restacked
	Steps           []MergePlanStep   // Ordered steps to execute
	Warnings        []string          // Non-blocking warnings
	CreatedAt       time.Time
}

// MergePlanValidation contains validation results
type MergePlanValidation struct {
	Valid    bool
	Errors   []string // Blocking errors
	Warnings []string // Non-blocking warnings
}

// CreateMergePlanOptions are options for creating a merge plan
type CreateMergePlanOptions struct {
	Strategy MergeStrategy
	Force    bool
	Engine   engine.Engine
	Splog    *tui.Splog
	RepoRoot string
}

// CreateMergePlan analyzes the current state and builds a merge plan
func CreateMergePlan(opts CreateMergePlanOptions) (*MergePlan, *MergePlanValidation, error) {
	eng := opts.Engine
	splog := opts.Splog

	// 1. Get current branch, validate not on trunk
	currentBranch := eng.CurrentBranch()
	if currentBranch == "" {
		return nil, nil, fmt.Errorf("not on a branch")
	}

	if eng.IsTrunk(currentBranch) {
		return nil, nil, fmt.Errorf("cannot merge from trunk. You must be on a branch that has a PR")
	}

	// Check if current branch is tracked
	if !eng.IsBranchTracked(currentBranch) {
		return nil, nil, fmt.Errorf("current branch %s is not tracked by stackit", currentBranch)
	}

	// 2. Collect branches from trunk to current
	scope := engine.Scope{RecursiveParents: true}
	parentBranches := eng.GetRelativeStack(currentBranch, scope)

	// Build full list: parent branches + current branch
	// Filter out trunk (it shouldn't be in the list, but be safe)
	allBranches := make([]string, 0, len(parentBranches)+1)
	trunk := eng.Trunk()
	for _, branchName := range parentBranches {
		if !eng.IsTrunk(branchName) {
			allBranches = append(allBranches, branchName)
		}
	}
	allBranches = append(allBranches, currentBranch)
	_ = trunk // Reserved for future use

	// 3. For each branch: fetch PR info, check status, CI checks
	branchesToMerge := []BranchMergeInfo{}
	validation := &MergePlanValidation{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	for _, branchName := range allBranches {
		// Get PR info
		prInfo, err := eng.GetPrInfo(branchName)
		if err != nil {
			splog.Debug("Failed to get PR info for %s: %v", branchName, err)
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Failed to get PR info for %s: %v", branchName, err))
			continue
		}

		// Check if PR exists
		if prInfo == nil || prInfo.Number == nil {
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Branch %s has no associated PR", branchName))
			continue
		}

		// Check PR state
		state := prInfo.State
		if state != "OPEN" {
			if state == "MERGED" {
				splog.Debug("Skipping %s: PR #%d is already merged", branchName, *prInfo.Number)
				continue
			}
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Branch %s PR #%d is %s (not open)", branchName, *prInfo.Number, state))
			continue
		}

		// Check if draft
		if prInfo.IsDraft && !opts.Force {
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Branch %s PR #%d is a draft", branchName, *prInfo.Number))
		}

		// Check if local matches remote
		// Only check if branch has a PR (meaning it exists on remote)
		// If branch doesn't have a PR or doesn't exist on remote, don't warn
		matchesRemote, err := eng.BranchMatchesRemote(branchName)
		if err != nil {
			splog.Debug("Failed to check if branch matches remote: %v", err)
			matchesRemote = true // Assume matches if check fails
		}
		// Only warn if branch has a PR (exists on remote) and differs
		// If branch doesn't exist on remote, that's fine (might be local-only or already merged)
		if !matchesRemote && prInfo != nil && prInfo.Number != nil {
			// Get detailed difference information
			diffInfo := getBranchRemoteDifference(branchName, eng, splog)
			if diffInfo != "" {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s differs from remote: %s", branchName, diffInfo))
			} else {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s differs from remote", branchName))
			}
		}

		// Get CI check status
		checksStatus := ChecksNone
		passing, pending, err := git.GetPRChecksStatus(branchName)
		if err != nil {
			splog.Debug("Failed to get PR checks status for %s: %v", branchName, err)
			// Don't fail on check status errors, just mark as none
		} else if pending {
			checksStatus = ChecksPending
			if !opts.Force {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s PR #%d has pending CI checks", branchName, *prInfo.Number))
			}
		} else if !passing {
			checksStatus = ChecksFailing
			if !opts.Force {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Branch %s PR #%d has failing CI checks", branchName, *prInfo.Number))
			}
		} else {
			checksStatus = ChecksPassing
		}

		branchesToMerge = append(branchesToMerge, BranchMergeInfo{
			BranchName:    branchName,
			PRNumber:      *prInfo.Number,
			PRURL:         prInfo.URL,
			IsDraft:       prInfo.IsDraft,
			ChecksStatus:  checksStatus,
			MatchesRemote: matchesRemote,
		})
	}

	// If no PRs to merge, return early
	if len(branchesToMerge) == 0 {
		return nil, validation, fmt.Errorf("no open PRs found to merge")
	}

	// 4. Detect branching stacks (siblings)
	// Check if current branch has siblings
	currentChildren := eng.GetChildren(currentBranch)
	if len(currentChildren) > 0 {
		validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s has %d child branch(es) that will be restacked: %v", currentBranch, len(currentChildren), currentChildren))
	}

	// 5. Identify upstack branches that need restacking
	upstackBranches := []string{}
	upstackScope := engine.Scope{RecursiveChildren: true}
	upstack := eng.GetRelativeStackUpstack(currentBranch)
	for _, branchName := range upstack {
		if eng.IsBranchTracked(branchName) {
			upstackBranches = append(upstackBranches, branchName)
		}
	}
	_ = upstackScope // Reserved for future use

	// 6. Build ordered steps based on strategy
	steps := []MergePlanStep{}
	if opts.Strategy == MergeStrategyTopDown {
		steps = buildTopDownSteps(branchesToMerge, currentBranch, upstackBranches, eng)
	} else {
		// Default to bottom-up
		steps = buildBottomUpSteps(branchesToMerge, currentBranch, upstackBranches, eng)
	}

	plan := &MergePlan{
		Strategy:        opts.Strategy,
		CurrentBranch:   currentBranch,
		BranchesToMerge: branchesToMerge,
		UpstackBranches: upstackBranches,
		Steps:           steps,
		Warnings:        validation.Warnings,
		CreatedAt:       time.Now(),
	}

	return plan, validation, nil
}

// buildBottomUpSteps builds steps for bottom-up merge strategy
func buildBottomUpSteps(branchesToMerge []BranchMergeInfo, currentBranch string, upstackBranches []string, eng engine.Engine) []MergePlanStep {
	steps := []MergePlanStep{}

	for i, branchInfo := range branchesToMerge {
		// 1. Merge PR
		steps = append(steps, MergePlanStep{
			StepType:    StepMergePR,
			BranchName:  branchInfo.BranchName,
			PRNumber:    branchInfo.PRNumber,
			Description: fmt.Sprintf("Merge PR #%d (%s)", branchInfo.PRNumber, branchInfo.BranchName),
		})

		// 2. Pull trunk
		steps = append(steps, MergePlanStep{
			StepType:    StepPullTrunk,
			BranchName:  "",
			PRNumber:    0,
			Description: "Pull trunk to get merged changes",
		})

		// 3. If not the last branch, restack the next one
		if i < len(branchesToMerge)-1 {
			nextBranch := branchesToMerge[i+1].BranchName
			steps = append(steps, MergePlanStep{
				StepType:    StepRestack,
				BranchName:  nextBranch,
				PRNumber:    0,
				Description: fmt.Sprintf("Restack %s onto trunk", nextBranch),
			})
		}
	}

	// 4. Delete all merged branches
	for _, branchInfo := range branchesToMerge {
		steps = append(steps, MergePlanStep{
			StepType:    StepDeleteBranch,
			BranchName:  branchInfo.BranchName,
			PRNumber:    0,
			Description: fmt.Sprintf("Delete local branch %s", branchInfo.BranchName),
		})
	}

	// 5. Restack upstack branches
	for _, upstackBranch := range upstackBranches {
		steps = append(steps, MergePlanStep{
			StepType:    StepRestack,
			BranchName:  upstackBranch,
			PRNumber:    0,
			Description: fmt.Sprintf("Restack %s onto trunk", upstackBranch),
		})
	}

	return steps
}

// buildTopDownSteps builds steps for top-down merge strategy
func buildTopDownSteps(branchesToMerge []BranchMergeInfo, currentBranch string, upstackBranches []string, eng engine.Engine) []MergePlanStep {
	steps := []MergePlanStep{}

	if len(branchesToMerge) == 0 {
		return steps
	}

	// Find the current branch in the list
	currentBranchInfo := branchesToMerge[len(branchesToMerge)-1]

	// 1. Rebase current branch onto trunk (squashing intermediate branches)
	// This is a complex operation that will be handled in execution
	steps = append(steps, MergePlanStep{
		StepType:    StepUpdatePRBase,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Rebase %s onto trunk (squashing %d intermediate branch(es))", currentBranch, len(branchesToMerge)-1),
	})

	// 2. Update PR base to trunk
	steps = append(steps, MergePlanStep{
		StepType:    StepUpdatePRBase,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Update PR #%d base branch to trunk", currentBranchInfo.PRNumber),
	})

	// 3. Merge the single PR
	steps = append(steps, MergePlanStep{
		StepType:    StepMergePR,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Merge PR #%d (%s)", currentBranchInfo.PRNumber, currentBranch),
	})

	// 4. Pull trunk
	steps = append(steps, MergePlanStep{
		StepType:    StepPullTrunk,
		BranchName:  "",
		PRNumber:    0,
		Description: "Pull trunk to get merged changes",
	})

	// 5. Delete intermediate branches (all except current)
	for _, branchInfo := range branchesToMerge[:len(branchesToMerge)-1] {
		steps = append(steps, MergePlanStep{
			StepType:    StepDeleteBranch,
			BranchName:  branchInfo.BranchName,
			PRNumber:    0,
			Description: fmt.Sprintf("Delete local branch %s", branchInfo.BranchName),
		})
	}

	// 6. Delete current branch
	steps = append(steps, MergePlanStep{
		StepType:    StepDeleteBranch,
		BranchName:  currentBranch,
		PRNumber:    0,
		Description: fmt.Sprintf("Delete local branch %s", currentBranch),
	})

	// 7. Restack upstack branches
	for _, upstackBranch := range upstackBranches {
		steps = append(steps, MergePlanStep{
			StepType:    StepRestack,
			BranchName:  upstackBranch,
			PRNumber:    0,
			Description: fmt.Sprintf("Restack %s onto trunk", upstackBranch),
		})
	}

	return steps
}

// FormatMergePlan formats a merge plan for display
func FormatMergePlan(plan *MergePlan, validation *MergePlanValidation) string {
	var result string

	result += fmt.Sprintf("Merge Strategy: %s\n", plan.Strategy)
	result += fmt.Sprintf("Current Branch: %s\n", plan.CurrentBranch)
	result += "\n"

	if len(plan.BranchesToMerge) > 0 {
		result += "Branches to merge (bottom to top):\n"
		for i, branchInfo := range plan.BranchesToMerge {
			marker := ""
			if branchInfo.BranchName == plan.CurrentBranch {
				marker = " ← current"
			}
			checksIcon := "✓"
			if branchInfo.ChecksStatus == ChecksFailing {
				checksIcon = "✗"
			} else if branchInfo.ChecksStatus == ChecksPending {
				checksIcon = "⏳"
			}
			result += fmt.Sprintf("  %d. %s  PR #%d  %s Checks %s%s\n", i+1, branchInfo.BranchName, branchInfo.PRNumber, checksIcon, branchInfo.ChecksStatus, marker)
		}
		result += "\n"
	}

	if len(plan.UpstackBranches) > 0 {
		result += "Branches above (will be restacked on trunk):\n"
		for _, branchName := range plan.UpstackBranches {
			result += fmt.Sprintf("  • %s\n", branchName)
		}
		result += "\n"
	}

	if len(validation.Errors) > 0 {
		result += "Errors:\n"
		for _, err := range validation.Errors {
			result += fmt.Sprintf("  ✗ %s\n", err)
		}
		result += "\n"
	}

	if len(validation.Warnings) > 0 {
		result += "Warnings:\n"
		for _, warn := range validation.Warnings {
			result += fmt.Sprintf("  ⚠ %s\n", warn)
		}
		result += "\n"
	}

	result += "Merge Plan:\n"
	for i, step := range plan.Steps {
		result += fmt.Sprintf("  %d. %s\n", i+1, step.Description)
	}

	return result
}

// getBranchRemoteDifference returns a detailed description of how a branch differs from remote
func getBranchRemoteDifference(branchName string, eng engine.Engine, splog *tui.Splog) string {
	// Get local SHA
	localSha, err := git.GetRevision(branchName)
	if err != nil {
		splog.Debug("Failed to get local SHA for %s: %v", branchName, err)
		return ""
	}

	// Get remote SHA - try from remote tracking branch first, then fall back to ls-remote
	remoteSha, err := git.GetRemoteRevision(branchName)
	if err != nil {
		// Remote tracking branch doesn't exist locally, try fetching directly from remote
		splog.Debug("Remote tracking branch not found for %s, fetching from remote: %v", branchName, err)
		remoteShas, err := git.FetchRemoteShas("origin")
		if err != nil {
			splog.Debug("Failed to fetch remote SHAs: %v", err)
			localShort := localSha
			if len(localSha) > 7 {
				localShort = localSha[:7]
			}
			return fmt.Sprintf("local: %s (unable to fetch remote SHA)", localShort)
		}
		var exists bool
		remoteSha, exists = remoteShas[branchName]
		if !exists {
			// Branch doesn't exist on remote
			localShort := localSha
			if len(localSha) > 7 {
				localShort = localSha[:7]
			}
			return fmt.Sprintf("local: %s (branch not found on remote)", localShort)
		}
	}

	if localSha == remoteSha {
		return "" // Shouldn't happen if we got here, but be safe
	}

	// Shorten SHAs for display
	localShort := localSha
	if len(localSha) > 7 {
		localShort = localSha[:7]
	}
	remoteShort := remoteSha
	if len(remoteSha) > 7 {
		remoteShort = remoteSha[:7]
	}

	// Try to determine relationship using git merge-base
	// Use remote tracking branch reference
	remoteBranchRef := "refs/remotes/origin/" + branchName
	commonAncestor, err := git.GetMergeBaseByRef(branchName, remoteBranchRef)
	if err != nil {
		// Can't determine relationship, just show SHAs
		// Most common case: local is ahead (has unpushed commits)
		return fmt.Sprintf("local: %s, remote: %s (likely local is ahead)", localShort, remoteShort)
	}

	if commonAncestor == localSha {
		// Local is behind remote
		return fmt.Sprintf("local is behind remote (local: %s, remote: %s)", localShort, remoteShort)
	} else if commonAncestor == remoteSha {
		// Local is ahead of remote
		return fmt.Sprintf("local is ahead of remote (local: %s, remote: %s)", localShort, remoteShort)
	} else {
		// Diverged
		return fmt.Sprintf("local and remote have diverged (local: %s, remote: %s)", localShort, remoteShort)
	}
}
