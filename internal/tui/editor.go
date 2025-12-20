package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// OpenEditor opens the user's preferred editor with the given initial content.
// It returns the edited content or an error.
func OpenEditor(initialContent, filenamePattern string) (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", filenamePattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write initial content
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get editor from environment
	editor := os.Getenv("GIT_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		// Try to get from git config
		output, err := exec.Command("git", "config", "--get", "core.editor").Output()
		if err == nil {
			editor = strings.TrimSpace(string(output))
		}
	}
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Open editor
	cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, tmpFile.Name()))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return string(content), nil
}
