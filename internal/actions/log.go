package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// LogOptions specifies options for the log command
type LogOptions struct {
	Style         string // "SHORT" or "FULL"
	Reverse       bool
	Steps         *int
	BranchName    string
	ShowUntracked bool
}

// LogAction displays the branch tree
func LogAction(opts LogOptions, ctx *runtime.Context) error {
	// Populate remote SHAs if needed
	if err := ctx.Engine.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	// Create tree renderer
	renderer := output.NewStackTreeRenderer(
		ctx.Engine.CurrentBranch(),
		ctx.Engine.Trunk(),
		ctx.Engine.GetChildren,
		ctx.Engine.GetParent,
		ctx.Engine.IsTrunk,
		ctx.Engine.IsBranchFixed,
	)

	// Render the stack
	stackLines := renderer.RenderStack(opts.BranchName, output.TreeRenderOptions{
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
	for _, branchName := range ctx.Engine.AllBranchNames() {
		if !ctx.Engine.IsTrunk(branchName) && !ctx.Engine.IsBranchTracked(branchName) {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}
