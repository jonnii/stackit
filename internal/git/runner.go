package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunGitCommand executes a git command and returns the output
func RunGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
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
