package create

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

func getCommitMessage(ctx context.Context) (string, error) {
	template, err := git.GetCommitTemplate(ctx)
	if err != nil {
		return "", err
	}

	msg, err := tui.OpenEditor(template, "COMMIT_EDITMSG-*")
	if err != nil {
		return "", err
	}

	return utils.CleanCommitMessage(msg), nil
}

// getCommitMessageForBranch gets the commit message needed for branch name generation.
// If branch name is not provided and commit message is empty, it will prompt for one in interactive mode.
func getCommitMessageForBranch(ctx *runtime.Context, opts *Options, commitMessage string) (string, error) {
	// If branch name is provided, we don't need commit message for branch generation
	if opts.BranchName != "" {
		return commitMessage, nil
	}

	// If commit message is empty, we need to get it
	if commitMessage == "" {
		if !utils.IsInteractive() {
			return "", fmt.Errorf("must specify either a branch name or commit message")
		}

		// Interactive: get commit message from editor
		msg, err := getCommitMessage(ctx.Context)
		if err != nil {
			return "", err
		}
		if msg == "" {
			return "", fmt.Errorf("aborting due to empty commit message")
		}
		return msg, nil
	}

	return commitMessage, nil
}
