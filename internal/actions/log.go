package actions

import (
	"strings"
	"sync"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// LogOptions contains options for the log command
type LogOptions struct {
	Style         string // "NORMAL" or "FULL"
	Reverse       bool
	Steps         *int
	BranchName    string
	ShowUntracked bool
}

// LogAction displays the branch tree
func LogAction(ctx *runtime.Context, opts LogOptions) error {
	// Populate remote SHAs if needed (only for FULL mode)
	if opts.Style == "FULL" {
		if err := ctx.Engine.PopulateRemoteShas(); err != nil {
			ctx.Splog.Debug("Failed to populate remote SHAs: %v", err)
		}
	}

	// Create tree renderer
	currentBranch := ctx.Engine.CurrentBranch()
	trunk := ctx.Engine.Trunk()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.Name
	}
	renderer := tui.NewStackTreeRenderer(
		currentBranchName,
		trunk.Name,
		func(branchName string) []string {
			branch := ctx.Engine.GetBranch(branchName)
			children := branch.GetChildren()
			childNames := make([]string, len(children))
			for i, c := range children {
				childNames[i] = c.Name
			}
			return childNames
		},
		func(branchName string) string {
			branch := ctx.Engine.GetBranch(branchName)
			parent := ctx.Engine.GetParent(branch)
			if parent == nil {
				return ""
			}
			return parent.Name
		},
		func(branchName string) bool { return ctx.Engine.GetBranch(branchName).IsTrunk() },
		func(branchName string) bool {
			return ctx.Engine.GetBranch(branchName).IsBranchUpToDate()
		},
	)

	// Render the stack
	// First, collect annotations for all branches in the stack
	annotations := make(map[string]tui.BranchAnnotation)
	allBranches := ctx.Engine.AllBranches()

	type result struct {
		branchName string
		annotation tui.BranchAnnotation
	}
	results := make(chan result, len(allBranches))
	var wg sync.WaitGroup

	for _, branch := range allBranches {
		wg.Add(1)
		go func(bName string) {
			defer wg.Done()
			branchObj := ctx.Engine.GetBranch(bName)
			annotation := tui.BranchAnnotation{
				Scope:         ctx.Engine.GetScopeInternal(bName).String(),
				ExplicitScope: ctx.Engine.GetExplicitScopeInternal(bName).String(),
			}

			// Local stats (always fast enough)
			if !branchObj.IsTrunk() {
				if count, err := branchObj.GetCommitCount(); err == nil {
					annotation.CommitCount = count
				}
				if added, deleted, err := branchObj.GetDiffStats(); err == nil {
					annotation.LinesAdded = added
					annotation.LinesDeleted = deleted
				}
			}

			// PR info (local metadata)
			if !branchObj.IsTrunk() {
				prInfo, _ := ctx.Engine.GetPrInfo(bName)
				if prInfo != nil {
					annotation.PRNumber = prInfo.Number
					annotation.PRState = prInfo.State
					annotation.IsDraft = prInfo.IsDraft
				}
			}

			// CI status (only in FULL mode)
			if opts.Style == "FULL" && !branchObj.IsTrunk() && ctx.GitHubClient != nil {
				if status, err := ctx.GitHubClient.GetPRChecksStatus(ctx.Context, bName); err == nil && status != nil {
					annotation.CheckStatus = "PASSING"
					if status.Pending {
						annotation.CheckStatus = "PENDING"
					} else if !status.Passing {
						annotation.CheckStatus = "FAILING"
					}
				}
			}

			results <- result{bName, annotation}
		}(branch.Name)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		annotations[res.branchName] = res.annotation
	}

	renderer.SetAnnotations(annotations)

	stackLines := renderer.RenderStack(opts.BranchName, tui.TreeRenderOptions{
		Short:   false, // We want the full tree characters with stats
		Reverse: opts.Reverse,
		Steps:   opts.Steps,
	})

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchNames(ctx)
		if len(untracked) > 0 {
			stackLines = append(stackLines, "")
			stackLines = append(stackLines, "Untracked branches:")
			stackLines = append(stackLines, untracked...)
		}
	}

	// Output the result
	ctx.Splog.Page(strings.Join(stackLines, "\n"))
	ctx.Splog.Newline()

	return nil
}

func getUntrackedBranchNames(ctx *runtime.Context) []string {
	var untracked []string
	for _, branch := range ctx.Engine.AllBranches() {
		branchName := branch.Name
		if !branch.IsTrunk() && !branch.IsTracked() {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}
