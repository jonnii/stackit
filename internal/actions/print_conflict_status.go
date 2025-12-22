package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// PrintConflictStatus displays conflict information and instructions to the user
func PrintConflictStatus(ctx context.Context, branchName string, splog *tui.Splog) error {
	msg := tui.ColorRed(fmt.Sprintf("Hit conflict restacking %s", branchName))
	splog.Info("%s", msg)
	splog.Newline()

	// Get unmerged files
	unmergedFiles, err := git.GetUnmergedFiles(ctx)
	if err == nil && len(unmergedFiles) > 0 {
		splog.Info("%s", tui.ColorYellow("Unmerged files:"))
		for _, file := range unmergedFiles {
			splog.Info("%s", tui.ColorRed(file))
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
		msg := tui.ColorYellow(fmt.Sprintf("You are here (resolving %s):", rebaseHeadShort))
		splog.Info("%s", msg)
		// Could show log here if needed
		splog.Newline()
	}

	splog.Info("%s", tui.ColorYellow("To fix and continue your previous Stackit command:"))
	splog.Info("(1) resolve the listed merge conflicts")
	splog.Info("(2) mark them as resolved with %s", tui.ColorCyan("stackit add ."))
	splog.Info("(3) run %s to continue executing your previous Stackit command", tui.ColorCyan("stackit continue"))
	splog.Info("It's safe to cancel the ongoing rebase with %s.", tui.ColorCyan("git rebase --abort"))

	return nil
}
