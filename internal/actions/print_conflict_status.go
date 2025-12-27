package actions

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// PrintConflictStatus displays conflict information and instructions to the user
func PrintConflictStatus(ctx context.Context, branchName string, splog *tui.Splog) error {
	msg := style.ColorRed(fmt.Sprintf("Hit conflict restacking %s", branchName))
	splog.Info("%s", msg)
	splog.Newline()

	// Get unmerged files
	unmergedFiles, err := git.GetUnmergedFiles(ctx)
	if err == nil && len(unmergedFiles) > 0 {
		splog.Info("%s", style.ColorYellow("Unmerged files:"))
		for _, file := range unmergedFiles {
			splog.Info("%s", style.ColorRed(file))
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
		msg := style.ColorYellow(fmt.Sprintf("You are here (resolving %s):", rebaseHeadShort))
		splog.Info("%s", msg)
		// Could show log here if needed
		splog.Newline()
	}

	splog.Info("%s", style.ColorYellow("To fix and continue your previous Stackit command:"))
	splog.Info("(1) resolve the listed merge conflicts")
	splog.Info("(2) mark them as resolved with %s", style.ColorCyan("stackit add ."))
	splog.Info("(3) run %s to continue executing your previous Stackit command", style.ColorCyan("stackit continue"))
	splog.Info("It's safe to cancel the ongoing rebase with %s.", style.ColorCyan("git rebase --abort"))

	return nil
}
