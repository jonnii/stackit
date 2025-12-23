package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// LogOptions contains options for the log command
type LogOptions struct {
	Style         string // "SHORT" or "FULL"
	Reverse       bool
	Steps         *int
	BranchName    string
	ShowUntracked bool
}

// LogAction displays the branch tree
func LogAction(ctx *runtime.Context, opts LogOptions) error {
	// Populate remote SHAs if needed
	if err := ctx.Engine.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	// Create tree renderer
	currentBranch := ctx.Engine.CurrentBranch()
	trunk := ctx.Engine.Trunk()
	renderer := tui.NewStackTreeRenderer(
		currentBranch.Name,
		trunk.Name,
		ctx.Engine.GetChildren,
		ctx.Engine.GetParent,
		func(branchName string) bool { return ctx.Engine.GetBranch(branchName).IsTrunk() },
		func(branchName string) bool {
			return ctx.Engine.IsBranchUpToDate(branchName)
		},
	)

	// Render the stack
	stackLines := renderer.RenderStack(opts.BranchName, tui.TreeRenderOptions{
		Short:   opts.Style == "SHORT",
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
