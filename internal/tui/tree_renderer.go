package tui

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/components/tree"
)

// NewStackTreeRenderer creates a tree renderer configured for the current engine state
func NewStackTreeRenderer(eng engine.BranchReader) *tree.StackTreeRenderer {
	currentBranch := eng.CurrentBranch()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.GetName()
	}

	trunk := eng.Trunk()

	return tree.NewStackTreeRenderer(
		currentBranchName,
		trunk.GetName(),
		func(branchName string) []string {
			branch := eng.GetBranch(branchName)
			children := branch.GetChildren()
			childNames := make([]string, len(children))
			for i, c := range children {
				childNames[i] = c.GetName()
			}
			return childNames
		},
		func(branchName string) string {
			branch := eng.GetBranch(branchName)
			parent := eng.GetParent(branch)
			if parent == nil {
				return ""
			}
			return parent.GetName()
		},
		func(branchName string) bool { return eng.IsTrunkInternal(branchName) },
		func(branchName string) bool {
			return eng.IsBranchUpToDateInternal(branchName)
		},
	)
}
