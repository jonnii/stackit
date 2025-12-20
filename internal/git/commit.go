package git

import (
	"context"
	"fmt"
	"strings"
)

// CommitOptions contains options for creating a commit
type CommitOptions struct {
	Message     string
	Amend       bool
	NoEdit      bool
	Edit        bool
	Verbose     int
	ResetAuthor bool
}

// Commit creates a commit with the given message
// If verbose > 0, shows unified diff in commit message template
func Commit(message string, verbose int) error {
	return CommitWithOptions(CommitOptions{
		Message: message,
		Verbose: verbose,
	})
}

// CommitWithOptions creates a commit with the given options
func CommitWithOptions(opts CommitOptions) error {
	args := []string{"commit"}

	if opts.Amend {
		args = append(args, "--amend")
	}

	if opts.ResetAuthor {
		args = append(args, "--reset-author")
	}

	if opts.Verbose > 0 {
		args = append(args, "-v")
	}

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	if opts.NoEdit {
		args = append(args, "--no-edit")
	} else if opts.Edit {
		// Only add -e if explicitly requested (git opens editor by default if no message)
		args = append(args, "-e")
	}
	// If neither NoEdit nor Edit is set, and no message is provided,
	// git will open the editor by default (no flag needed)

	return RunGitCommandInteractive(args...)
}

// GetStagedDiff returns the unified diff of staged changes
func GetStagedDiff(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff", "--cached"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	output, err := RunGitCommandRawWithContext(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return output, nil
}

// GetCommitTemplate returns a string suitable for a commit message template
func GetCommitTemplate(ctx context.Context) (string, error) {
	status, err := RunGitCommandWithContext(ctx, "status")
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(status, "\n")
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("# Please enter the commit message for your changes. Lines starting\n")
	sb.WriteString("# with '#' will be ignored, and an empty message aborts the commit.\n")
	sb.WriteString("#\n")
	for _, line := range lines {
		sb.WriteString("# ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GetUnstagedDiff returns the unified diff of unstaged changes
func GetUnstagedDiff(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	output, err := RunGitCommandRawWithContext(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged diff: %w", err)
	}
	return output, nil
}
