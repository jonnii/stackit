package git

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"stackit.dev/stackit/internal/errors"
)

// DefaultCommandTimeout is the default timeout for git commands
const DefaultCommandTimeout = 5 * time.Minute

// RunGitCommand executes a git command and returns the output
// It uses context.Background() with a default timeout.
func RunGitCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	return RunGitCommandWithContext(ctx, args...)
}

// RunGitCommandWithContext executes a git command with the given context and returns the output
func RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
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
	return RunGitCommandWithInputAndContext(ctx, input, args...)
}

// RunGitCommandWithInputAndContext executes a git command with input and context, returning the output
func RunGitCommandWithInputAndContext(ctx context.Context, input string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin = strings.NewReader(input)
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
