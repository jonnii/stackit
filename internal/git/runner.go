package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"stackit.dev/stackit/internal/errors"
)

// DefaultCommandTimeout is the default timeout for git commands
const DefaultCommandTimeout = 5 * time.Minute

// workingDir is an optional working directory for git commands.
// When empty, commands run in the current working directory.
// This can be set via SetWorkingDir for test isolation.
var workingDir string

// SetWorkingDir sets the working directory for all git commands.
// Pass empty string to use the current working directory.
// This is primarily used for test isolation to enable parallel tests.
func SetWorkingDir(dir string) {
	workingDir = dir
}

// GetWorkingDir returns the current working directory setting.
func GetWorkingDir() string {
	return workingDir
}

// RunGitCommand executes a git command and returns the output
// It uses context.Background() with a default timeout.
func RunGitCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	return RunGitCommandWithContext(ctx, args...)
}

// RunGitCommandInDir executes a git command in a specific directory and returns the output.
func RunGitCommandInDir(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	return runGitCommandInternal(ctx, dir, "", args...)
}

// RunGitCommandWithContext executes a git command with the given context and returns the output
func RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	return runGitCommandInternal(ctx, workingDir, "", args...)
}

// runGitCommandInternal is the internal implementation that handles directory and input
func runGitCommandInternal(ctx context.Context, dir string, input string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
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
			return "", errors.NewGitCommandError("git", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", errors.NewGitCommandError("git", args, stdout.String(), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunGitCommandLines executes a git command and returns output as lines
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

// RunGitCommandWithInput executes a git command with input and returns the output
// It uses context.Background() with a default timeout.
func RunGitCommandWithInput(input string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	return runGitCommandInternal(ctx, workingDir, input, args...)
}

// RunGitCommandWithInputAndContext executes a git command with input and context, returning the output
func RunGitCommandWithInputAndContext(ctx context.Context, input string, args ...string) (string, error) {
	return runGitCommandInternal(ctx, workingDir, input, args...)
}

// RunGitCommandInteractive executes a git command interactively with stdin/stdout/stderr
// connected to the terminal. This is needed for commands like `git add -p` that require
// user interaction.
func RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
