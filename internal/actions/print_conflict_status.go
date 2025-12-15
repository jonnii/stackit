package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// PrintConflictStatus displays conflict information and instructions to the user
func PrintConflictStatus(branchName string, eng engine.Engine, splog *output.Splog) error {
	splog.Info(output.ColorRed(fmt.Sprintf("Hit conflict restacking %s", branchName)))
	splog.Newline()

	// Get unmerged files
	unmergedFiles, err := git.GetUnmergedFiles()
	if err == nil && len(unmergedFiles) > 0 {
		splog.Info(output.ColorYellow("Unmerged files:"))
		for _, file := range unmergedFiles {
			splog.Info(output.ColorRed(file))
		}
		splog.Newline()
	}

	// Get rebase head
	rebaseHead, err := git.GetRebaseHead()
	if err == nil && rebaseHead != "" {
		rebaseHeadShort := rebaseHead
		if len(rebaseHead) > 7 {
			rebaseHeadShort = rebaseHead[:7]
		}
		splog.Info(output.ColorYellow(fmt.Sprintf("You are here (resolving %s):", rebaseHeadShort)))
		// Could show log here if needed
		splog.Newline()
	}

	splog.Info(output.ColorYellow("To fix and continue your previous Stackit command:"))
	splog.Info("(1) resolve the listed merge conflicts")
	splog.Info("(2) mark them as resolved with %s", output.ColorCyan("stackit add ."))
	splog.Info("(3) run %s to continue executing your previous Stackit command", output.ColorCyan("stackit continue"))
	splog.Info("It's safe to cancel the ongoing rebase with %s.", output.ColorCyan("git rebase --abort"))

	return nil
}
