package git

import (
	"fmt"
	"os"
	"os/exec"
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

	// Use exec.Command directly to allow for interactive commit if needed
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// GetStagedDiff returns the unified diff of staged changes
func GetStagedDiff(files ...string) (string, error) {
	args := []string{"diff", "--cached"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	output, err := RunGitCommandRaw(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return output, nil
}

// GetUnstagedDiff returns the unified diff of unstaged changes
func GetUnstagedDiff(files ...string) (string, error) {
	args := []string{"diff"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	output, err := RunGitCommandRaw(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged diff: %w", err)
	}
	return output, nil
}
