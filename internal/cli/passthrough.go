package cli

import (
	"fmt"
	"os"
	"os/exec"
)

var gitCommandAllowlist = []string{
	"add",
	"am",
	"apply",
	"archive",
	"bisect",
	"blame",
	"bundle",
	"cherry-pick",
	"clean",
	"clone",
	"diff",
	"difftool",
	"fetch",
	"format-patch",
	"fsck",
	"grep",
	// "merge" removed - stackit has its own merge command
	"mv",
	"notes",
	"pull",
	"push",
	"range-diff",
	"rebase",
	"reflog",
	"remote",
	"request-pull",
	"reset",
	"restore",
	"revert",
	"rm",
	"show",
	"send-email",
	"sparse-checkout",
	"stash",
	"status",
	"submodule",
	"switch",
	"tag",
}

// HandlePassthrough checks if the command should be passed through to git
// and executes it if so. Returns true if the command was handled (and the program should exit).
func HandlePassthrough(args []string) bool {
	if len(args) < 2 {
		return false
	}

	command := args[1]
	if !contains(gitCommandAllowlist, command) {
		return false
	}

	// Build the git command
	gitArgs := args[1:]
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stdin = os.Stdin
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	// Print passthrough message
	fmt.Fprintf(os.Stderr, "\033[90mPassing command through to git...\033[0m\n")
	fmt.Fprintf(os.Stderr, "\033[90mRunning: \"git %s\"\033[0m\n\n", joinArgs(gitArgs))

	// Execute git command
	err := gitCmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}
