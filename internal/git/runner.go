// Package git provides a wrapper around git commands and go-git for repository operations.
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	stackiterrors "stackit.dev/stackit/internal/errors"
)

// DefaultCommandTimeout is the default timeout for git commands
const DefaultCommandTimeout = 5 * time.Minute

// ErrStaleRemoteInfo indicates that a push failed because the remote has changed
var ErrStaleRemoteInfo = errors.New("stale info")

// Runner handles execution of git commands
type Runner struct {
	workingDir string
}

// NewRunner creates a new Runner
func NewRunner(workingDir string) *Runner {
	return &Runner{workingDir: workingDir}
}

// defaultRunner is the global runner used by the package-level functions
var defaultRunner = &Runner{}

// SetWorkingDir sets the working directory for the default git runner.
func SetWorkingDir(dir string) {
	defaultRunner.workingDir = dir
}

// GetWorkingDir returns the current working directory setting for the default runner.
func GetWorkingDir() string {
	return defaultRunner.workingDir
}

// RunGitCommand executes a git command using the default runner and returns the output.
// It uses context.Background() with a default timeout.
func RunGitCommand(args ...string) (string, error) {
	return defaultRunner.Run(context.Background(), args...)
}

// RunGitCommandInDir executes a git command in a specific directory and returns the output.
func RunGitCommandInDir(dir string, args ...string) (string, error) {
	runner := &Runner{workingDir: dir}
	return runner.Run(context.Background(), args...)
}

// RunGitCommandWithContext executes a git command with the given context using the default runner.
func RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	return defaultRunner.Run(ctx, args...)
}

// Run executes a git command with the given context and returns the output
func (r *Runner) Run(ctx context.Context, args ...string) (string, error) {
	return r.runInternal(ctx, "", true, args...)
}

// runInternal is the internal implementation that handles directory and input
func (r *Runner) runInternal(ctx context.Context, input string, trim bool, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// If no timeout/deadline is set in the context, add the default one
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if r.workingDir != "" {
		cmd.Dir = r.workingDir
	}
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", stackiterrors.NewGitCommandError("git", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", stackiterrors.NewGitCommandError("git", args, stdout.String(), stderr.String(), err)
	}
	if trim {
		return strings.TrimSpace(stdout.String()), nil
	}
	return stdout.String(), nil
}

// RunGitCommandRaw executes a git command using the default runner and returns the raw output (no trimming)
func RunGitCommandRaw(args ...string) (string, error) {
	return defaultRunner.runInternal(context.Background(), "", false, args...)
}

// RunGitCommandRawWithContext executes a git command using the default runner and returns the raw output (no trimming) with context
func RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error) {
	return defaultRunner.runInternal(ctx, "", false, args...)
}

// RunGitCommandLines executes a git command using the default runner and returns output as lines
func RunGitCommandLines(args ...string) ([]string, error) {
	output, err := RunGitCommand(args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// RunGitCommandLinesWithContext executes a git command with context and returns output as lines
func RunGitCommandLinesWithContext(ctx context.Context, args ...string) ([]string, error) {
	output, err := RunGitCommandWithContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// RunGitCommandWithInput executes a git command with input using the default runner and returns the output
func RunGitCommandWithInput(input string, args ...string) (string, error) {
	return defaultRunner.runInternal(context.Background(), input, true, args...)
}

// RunGitCommandWithInputAndContext executes a git command with input and context using the default runner
func RunGitCommandWithInputAndContext(ctx context.Context, input string, args ...string) (string, error) {
	return defaultRunner.runInternal(ctx, input, true, args...)
}

// RunGHCommandWithContext executes a gh command with the given context.
func RunGHCommandWithContext(ctx context.Context, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// If no timeout/deadline is set in the context, add the default one
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	if defaultRunner.workingDir != "" {
		cmd.Dir = defaultRunner.workingDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", stackiterrors.NewGitCommandError("gh", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", stackiterrors.NewGitCommandError("gh", args, stdout.String(), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunGitCommandInteractive executes a git command interactively with stdin/stdout/stderr
// connected to the terminal.
func RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	if defaultRunner.workingDir != "" {
		cmd.Dir = defaultRunner.workingDir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
