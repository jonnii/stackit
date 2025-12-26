package actions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// ForeachOptions contains options for the foreach command
type ForeachOptions struct {
	Command  string
	Args     []string
	Scope    engine.StackRange
	FailFast bool
}

// ForeachAction executes a command on each branch in the stack
func ForeachAction(ctx *runtime.Context, opts ForeachOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	// Get branches based on scope
	branches := currentBranch.GetRelativeStack(opts.Scope)
	if len(branches) == 0 {
		return nil
	}

	originalBranchName := currentBranch.Name
	defer func() {
		// Always try to return to the original branch
		if err := git.CheckoutBranch(ctx.Context, originalBranchName); err != nil {
			splog.Error("Failed to return to original branch %s: %v", originalBranchName, err)
		}
	}()

	// Join command and args into a single string for shell execution
	fullCommand := strings.Join(append([]string{opts.Command}, opts.Args...), " ")

	for _, branch := range branches {
		// Skip trunk
		if branch.IsTrunk() {
			continue
		}

		isCurrent := branch.Name == originalBranchName
		splog.Info("\nRunning on branch %s...", tui.ColorBranchName(branch.Name, isCurrent))

		if err := git.CheckoutBranch(ctx.Context, branch.Name); err != nil {
			splog.Error("Failed to checkout %s: %v", branch.Name, err)
			if opts.FailFast {
				return err
			}
			continue
		}

		// Execute the command via shell
		cmd := exec.CommandContext(ctx.Context, "/bin/sh", "-c", fullCommand)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Dir = ctx.RepoRoot

		if err := cmd.Run(); err != nil {
			splog.Error("✗ Command failed on branch %s: %v", branch.Name, err)
			if opts.FailFast {
				return fmt.Errorf("command failed on branch %s", branch.Name)
			}
		} else {
			splog.Info("✓ Command succeeded on branch %s", branch.Name)
		}
	}

	return nil
}
