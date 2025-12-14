package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
)

// RestackBranches restacks a list of branches
func RestackBranches(branchNames []string, eng engine.Engine, splog *output.Splog) error {
	for _, branchName := range branchNames {
		if eng.IsTrunk(branchName) {
			splog.Info("%s does not need to be restacked.", output.ColorBranchName(branchName, false))
			continue
		}

		result, err := eng.RestackBranch(branchName)
		if err != nil {
			return fmt.Errorf("failed to restack %s: %w", branchName, err)
		}

		switch result {
		case engine.RestackDone:
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}
			splog.Info("Restacked %s on %s.", 
				output.ColorBranchName(branchName, true),
				output.ColorBranchName(parent, false))
		case engine.RestackConflict:
			return fmt.Errorf("hit conflict restacking %s", branchName)
		case engine.RestackUnneeded:
			parent := eng.GetParent(branchName)
			if parent == "" {
				parent = eng.Trunk()
			}
			splog.Info("%s does not need to be restacked on %s.",
				output.ColorBranchName(branchName, false),
				output.ColorBranchName(parent, false))
		}
	}

	return nil
}
