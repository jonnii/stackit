package utils

import (
	"io"
	"os"
	"strings"
)

// ReadFromStdin reads all content from standard input
func ReadFromStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	// If it's a terminal, we don't want to block waiting for input
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}

	// If it's a regular file and it's empty, return empty (don't block)
	if stat.Mode().IsRegular() && stat.Size() == 0 {
		return "", nil
	}

	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(bytes))
	return result, nil
}
