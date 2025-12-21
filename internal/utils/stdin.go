package utils

import (
	"io"
	"os"
	"strings"
)

// ReadFromStdin reads all content from standard input
func ReadFromStdin() (string, error) {
	// If it's a terminal, we don't want to block waiting for input
	if IsInteractive() {
		return "", nil
	}

	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(bytes)), nil
}
