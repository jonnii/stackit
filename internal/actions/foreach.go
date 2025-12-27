package actions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
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

	originalBranchName := currentBranch.GetName()
	defer func() {
		// Always try to return to the original branch
		if err := eng.CheckoutBranch(ctx.Context, eng.GetBranch(originalBranchName)); err != nil {
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

		isCurrent := branch.GetName() == originalBranchName
		splog.Info("\nRunning on branch %s...", style.ColorBranchName(branch.GetName(), isCurrent))

		if err := eng.CheckoutBranch(ctx.Context, branch); err != nil {
			splog.Error("Failed to checkout %s: %v", branch.GetName(), err)
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
			splog.Error("✗ Command failed on branch %s: %v", branch.GetName(), err)
			if opts.FailFast {
				return fmt.Errorf("command failed on branch %s", branch.GetName())
			}
		} else {
			splog.Info("✓ Command succeeded on branch %s", branch.GetName())
		}
	}

	return nil
}
