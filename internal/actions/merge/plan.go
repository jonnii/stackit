package merge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

// Strategy defines how PRs in the stack should be merged
type Strategy string

const (
	// StrategyBottomUp merges PRs from the bottom of the stack up to the current branch
	StrategyBottomUp Strategy = "bottom-up"
	// StrategyTopDown merges the entire stack into a single PR
	StrategyTopDown Strategy = "top-down"
)

// StepType represents the type of step in a merge plan
type StepType string

const (
	// StepMergePR represents merging a PR
	StepMergePR StepType = "MERGE_PR"
	// StepRestack represents restacking a branch onto its parent
	StepRestack StepType = "RESTACK"
	// StepDeleteBranch represents deleting a local branch
	StepDeleteBranch StepType = "DELETE_BRANCH"
	// StepUpdatePRBase represents updating a PR's base branch
	StepUpdatePRBase StepType = "UPDATE_PR_BASE"
	// StepPullTrunk represents pulling the trunk branch
	StepPullTrunk StepType = "PULL_TRUNK"
	// StepWaitCI represents waiting for CI checks to complete
	StepWaitCI StepType = "WAIT_CI"
)

// ChecksStatus represents the CI check status for a PR
type ChecksStatus string

const (
	// ChecksPassing indicates all checks passed
	ChecksPassing ChecksStatus = "PASSING"
	// ChecksFailing indicates at least one check failed
	ChecksFailing ChecksStatus = "FAILING"
	// ChecksPending indicates checks are still running
	ChecksPending ChecksStatus = "PENDING"
	// ChecksNone indicates no checks are configured
	ChecksNone ChecksStatus = "NONE"
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

// PlanStep represents a single step in the merge plan
type PlanStep struct {
	StepType    StepType
	BranchName  string
	PRNumber    int
	Description string        // Human-readable description for display
	WaitTimeout time.Duration // Timeout for waiting steps (e.g., CI checks)
}

// Plan is the complete plan for a merge operation
type Plan struct {
	Strategy        Strategy
	CurrentBranch   string
	BranchesToMerge []BranchMergeInfo // Branches that will be merged (bottom to top)
	UpstackBranches []string          // Branches above current that will be restacked
	Steps           []PlanStep        // Ordered steps to execute
	Warnings        []string          // Non-blocking warnings
	CreatedAt       time.Time
}

// PlanValidation contains validation results
type PlanValidation struct {
	Valid    bool
	Errors   []string // Blocking errors
	Warnings []string // Non-blocking warnings
}

// CreatePlanOptions contains options for creating a merge plan
type CreatePlanOptions struct {
	Strategy Strategy
	Force    bool
}

// mergePlanEngine is a minimal interface needed for creating a merge plan
type mergePlanEngine interface {
	engine.BranchReader
	engine.PRManager
	engine.SyncManager
}

// CreateMergePlan analyzes the current state and builds a merge plan
func CreateMergePlan(ctx context.Context, eng mergePlanEngine, splog *tui.Splog, githubClient github.Client, opts CreatePlanOptions) (*Plan, *PlanValidation, error) {
	// 1. Get current branch, validate not on trunk
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return nil, nil, fmt.Errorf("not on a branch")
	}

	if currentBranch.IsTrunk() {
		return nil, nil, fmt.Errorf("cannot merge from trunk. You must be on a branch that has a PR")
	}

	// Check if current branch is tracked
	if !currentBranch.IsTracked() {
		return nil, nil, fmt.Errorf("current branch %s is not tracked by stackit", currentBranch.Name)
	}

	// 2. Collect branches from trunk to current
	scope := engine.Scope{RecursiveParents: true}
	parentBranches := currentBranch.GetRelativeStack(scope)

	// Build full list: parent branches + current branch
	// Filter out trunk (it shouldn't be in the list, but be safe)
	allBranches := make([]string, 0, len(parentBranches)+1)
	for _, branch := range parentBranches {
		if !branch.IsTrunk() {
			allBranches = append(allBranches, branch.Name)
		}
	}
	allBranches = append(allBranches, currentBranch.Name)

	// 3. For each branch: fetch PR info, check status, CI checks
	branchesToMerge := []BranchMergeInfo{}
	validation := &PlanValidation{
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
		matchesRemote, err := eng.BranchMatchesRemote(branchName)
		if err != nil {
			splog.Debug("Failed to check if branch matches remote: %v", err)
			matchesRemote = true // Assume matches if check fails
		}
		if !matchesRemote && prInfo != nil && prInfo.Number != nil {
			// Get detailed difference information
			diffInfo := getBranchRemoteDifference(branchName, splog)
			if diffInfo != "" {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s differs from remote: %s", branchName, diffInfo))
			} else {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Branch %s differs from remote", branchName))
			}
		}

		// Get CI check status
		checksStatus := ChecksNone
		if githubClient != nil {
			status, checkErr := githubClient.GetPRChecksStatus(ctx, branchName)
			switch {
			case checkErr != nil:
				splog.Debug("Failed to get PR checks status for %s: %v", branchName, checkErr)
			case status.Pending:
				checksStatus = ChecksPending
			case !status.Passing:
				checksStatus = ChecksFailing
				if !opts.Force {
					validation.Valid = false
					validation.Errors = append(validation.Errors, fmt.Sprintf("Branch %s PR #%d has failing CI checks", branchName, *prInfo.Number))
				}
			default:
				checksStatus = ChecksPassing
			}
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
	mergedSet := make(map[string]bool)
	for _, branch := range allBranches {
		mergedSet[branch] = true
	}

	for _, ancestor := range allBranches {
		ancestorBranch := eng.GetBranch(ancestor)
		if ancestorBranch.IsTrunk() {
			continue
		}
		children := ancestorBranch.GetChildren()
		for _, child := range children {
			if !mergedSet[child.Name] {
				validation.Warnings = append(validation.Warnings, fmt.Sprintf("Sibling branch %s will be reparented to trunk when %s is merged", child, ancestor))
			}
		}
	}

	// 5. Identify upstack branches that need restacking
	upstackBranches := []string{}
	upstack := eng.GetRelativeStackUpstack(*currentBranch)
	for _, branch := range upstack {
		if branch.IsTracked() {
			upstackBranches = append(upstackBranches, branch.Name)
		}
	}

	// 6. Build ordered steps based on strategy
	var steps []PlanStep
	if opts.Strategy == StrategyTopDown {
		steps = buildTopDownSteps(branchesToMerge, currentBranch.Name, upstackBranches)
	} else {
		steps = buildBottomUpSteps(branchesToMerge, upstackBranches)
	}

	plan := &Plan{
		Strategy:        opts.Strategy,
		CurrentBranch:   currentBranch.Name,
		BranchesToMerge: branchesToMerge,
		UpstackBranches: upstackBranches,
		Steps:           steps,
		Warnings:        validation.Warnings,
		CreatedAt:       time.Now(),
	}

	return plan, validation, nil
}

func buildBottomUpSteps(branchesToMerge []BranchMergeInfo, upstackBranches []string) []PlanStep {
	steps := []PlanStep{}
	defaultTimeout := 10 * time.Minute

	for i, branchInfo := range branchesToMerge {
		steps = append(steps, PlanStep{
			StepType:    StepWaitCI,
			BranchName:  branchInfo.BranchName,
			PRNumber:    branchInfo.PRNumber,
			Description: fmt.Sprintf("Wait for CI checks on PR #%d (%s)", branchInfo.PRNumber, branchInfo.BranchName),
			WaitTimeout: defaultTimeout,
		})

		steps = append(steps, PlanStep{
			StepType:    StepMergePR,
			BranchName:  branchInfo.BranchName,
			PRNumber:    branchInfo.PRNumber,
			Description: fmt.Sprintf("Merge PR #%d (%s)", branchInfo.PRNumber, branchInfo.BranchName),
		})

		steps = append(steps, PlanStep{
			StepType:    StepPullTrunk,
			BranchName:  "",
			PRNumber:    0,
			Description: "Pull trunk to get merged changes",
		})

		if i < len(branchesToMerge)-1 {
			nextBranch := branchesToMerge[i+1].BranchName
			steps = append(steps, PlanStep{
				StepType:    StepRestack,
				BranchName:  nextBranch,
				PRNumber:    0,
				Description: fmt.Sprintf("Restack %s onto trunk", nextBranch),
			})
		}
	}

	for _, branchInfo := range branchesToMerge {
		steps = append(steps, PlanStep{
			StepType:    StepDeleteBranch,
			BranchName:  branchInfo.BranchName,
			PRNumber:    0,
			Description: fmt.Sprintf("Delete local branch %s", branchInfo.BranchName),
		})
	}

	for _, upstackBranch := range upstackBranches {
		steps = append(steps, PlanStep{
			StepType:    StepRestack,
			BranchName:  upstackBranch,
			PRNumber:    0,
			Description: fmt.Sprintf("Restack %s onto trunk", upstackBranch),
		})
	}

	return steps
}

func buildTopDownSteps(branchesToMerge []BranchMergeInfo, currentBranch string, upstackBranches []string) []PlanStep {
	steps := []PlanStep{}

	if len(branchesToMerge) == 0 {
		return steps
	}

	currentBranchInfo := branchesToMerge[len(branchesToMerge)-1]

	steps = append(steps, PlanStep{
		StepType:    StepUpdatePRBase,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Rebase %s onto trunk (squashing %d intermediate branch(es))", currentBranch, len(branchesToMerge)-1),
	})

	steps = append(steps, PlanStep{
		StepType:    StepUpdatePRBase,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Update PR #%d base branch to trunk", currentBranchInfo.PRNumber),
	})

	steps = append(steps, PlanStep{
		StepType:    StepWaitCI,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Wait for CI checks on PR #%d (%s)", currentBranchInfo.PRNumber, currentBranch),
		WaitTimeout: 10 * time.Minute,
	})

	steps = append(steps, PlanStep{
		StepType:    StepMergePR,
		BranchName:  currentBranch,
		PRNumber:    currentBranchInfo.PRNumber,
		Description: fmt.Sprintf("Merge PR #%d (%s)", currentBranchInfo.PRNumber, currentBranch),
	})

	steps = append(steps, PlanStep{
		StepType:    StepPullTrunk,
		BranchName:  "",
		PRNumber:    0,
		Description: "Pull trunk to get merged changes",
	})

	for _, branchInfo := range branchesToMerge[:len(branchesToMerge)-1] {
		steps = append(steps, PlanStep{
			StepType:    StepDeleteBranch,
			BranchName:  branchInfo.BranchName,
			PRNumber:    0,
			Description: fmt.Sprintf("Delete local branch %s", branchInfo.BranchName),
		})
	}

	steps = append(steps, PlanStep{
		StepType:    StepDeleteBranch,
		BranchName:  currentBranch,
		PRNumber:    0,
		Description: fmt.Sprintf("Delete local branch %s", currentBranch),
	})

	for _, upstackBranch := range upstackBranches {
		steps = append(steps, PlanStep{
			StepType:    StepRestack,
			BranchName:  upstackBranch,
			PRNumber:    0,
			Description: fmt.Sprintf("Restack %s onto trunk", upstackBranch),
		})
	}

	return steps
}

// FormatMergePlan returns a human-readable representation of a merge plan
func FormatMergePlan(plan *Plan, validation *PlanValidation) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Merge Strategy: %s\n", plan.Strategy))
	result.WriteString(fmt.Sprintf("Current Branch: %s\n", plan.CurrentBranch))
	result.WriteString("\n")

	if len(validation.Errors) > 0 {
		result.WriteString("Errors:\n")
		for _, err := range validation.Errors {
			result.WriteString(fmt.Sprintf("  ✗ %s\n", err))
		}
		result.WriteString("\n")
	}

	if len(validation.Warnings) > 0 {
		result.WriteString("Warnings:\n")
		for _, warn := range validation.Warnings {
			result.WriteString(fmt.Sprintf("  ⚠ %s\n", warn))
		}
		result.WriteString("\n")
	}

	result.WriteString("Merge Plan:\n")
	for i, step := range plan.Steps {
		result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step.Description))
	}

	return result.String()
}

func getBranchRemoteDifference(branchName string, splog *tui.Splog) string {
	localSha, err := git.GetRevision(branchName)
	if err != nil {
		splog.Debug("Failed to get local SHA for %s: %v", branchName, err)
		return ""
	}

	remoteSha, err := git.GetRemoteRevision(branchName)
	if err != nil {
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
			localShort := localSha
			if len(localSha) > 7 {
				localShort = localSha[:7]
			}
			return fmt.Sprintf("local: %s (branch not found on remote)", localShort)
		}
	}

	if localSha == remoteSha {
		return ""
	}

	localShort := localSha
	if len(localSha) > 7 {
		localShort = localSha[:7]
	}
	remoteShort := remoteSha
	if len(remoteSha) > 7 {
		remoteShort = remoteSha[:7]
	}

	remoteBranchRef := "refs/remotes/origin/" + branchName
	commonAncestor, err := git.GetMergeBaseByRef(branchName, remoteBranchRef)
	if err != nil {
		return fmt.Sprintf("local: %s, remote: %s (likely local is ahead)", localShort, remoteShort)
	}

	switch {
	case commonAncestor == localSha:
		return fmt.Sprintf("local is behind remote (local: %s, remote: %s)", localShort, remoteShort)
	case commonAncestor == remoteSha:
		return fmt.Sprintf("local is ahead of remote (local: %s, remote: %s)", localShort, remoteShort)
	default:
		return fmt.Sprintf("local and remote have diverged (local: %s, remote: %s)", localShort, remoteShort)
	}
}
