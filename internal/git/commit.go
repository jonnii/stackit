package git

import (
	"fmt"
	"os"
	"os/exec"
)

// CommitOptions contains options for creating a commit
type CommitOptions struct {
	Message string
	Amend   bool
	NoEdit  bool
	Edit    bool
	Verbose int
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

	if opts.Verbose > 0 {
		args = append(args, "-v")
	}

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	if opts.NoEdit {
		args = append(args, "--no-edit")
	} else if opts.Edit {
		args = append(args, "-e")
	}

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
func GetStagedDiff() (string, error) {
	output, err := RunGitCommand("diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return output, nil
}

// GetUnstagedDiff returns the unified diff of unstaged changes
func GetUnstagedDiff() (string, error) {
	output, err := RunGitCommand("diff")
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged diff: %w", err)
	}
	return output, nil
}
