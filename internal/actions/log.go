package actions

import (
	"strings"
	"sync"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
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
	renderer := tui.NewStackTreeRenderer(ctx.Engine)

	// Render the stack
	// First, collect annotations for all branches in the stack
	annotations := make(map[string]tree.BranchAnnotation)
	allBranches := ctx.Engine.AllBranches()

	type result struct {
		branchName string
		annotation tree.BranchAnnotation
	}
	results := make(chan result, len(allBranches))
	var wg sync.WaitGroup

	for _, branch := range allBranches {
		wg.Add(1)
		go func(bName string) {
			defer wg.Done()
			branchObj := ctx.Engine.GetBranch(bName)
			annotation := tree.BranchAnnotation{
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
				branch := ctx.Engine.GetBranch(bName)
				prInfo, _ := ctx.Engine.GetPrInfo(branch)
				if prInfo != nil {
					annotation.PRNumber = prInfo.Number()
					annotation.PRState = prInfo.State()
					annotation.IsDraft = prInfo.IsDraft()
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
		}(branch.GetName())
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		annotations[res.branchName] = res.annotation
	}

	renderer.SetAnnotations(annotations)

	stackLines := renderer.RenderStack(opts.BranchName, tree.RenderOptions{
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
		branchName := branch.GetName()
		if !branch.IsTrunk() && !branch.IsTracked() {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}
