// Package testhelper provides shared test utilities for CLI packages.
package testhelper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	sharedBinaryPath string
	binaryOnce       sync.Once
	binaryErr        error
)

// SetSharedBinaryPath sets the shared binary path for tests.
// This is called by TestMain in cli_test package.
func SetSharedBinaryPath(path string) {
	sharedBinaryPath = path
}

// GetSharedBinaryPath returns the shared binary path, building it if necessary.
// This function is safe to call from any test package and will build the binary
// lazily on first access if it hasn't been set via SetSharedBinaryPath.
func GetSharedBinaryPath() string {
	binaryOnce.Do(func() {
		if sharedBinaryPath == "" {
			// Build the binary lazily
			path, err := buildBinary()
			if err != nil {
				binaryErr = err
				return
			}
			sharedBinaryPath = path
		}
	})
	return sharedBinaryPath
}

// GetBinaryError returns any error that occurred during binary building.
func GetBinaryError() error {
	return binaryErr
}

// buildBinary builds the stackit binary and returns its path.
func buildBinary() (string, error) {
	// Find the module root by walking up from the current directory
	// looking for go.mod file
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	moduleRoot := findModuleRoot(wd)
	if moduleRoot == "" {
		return "", fmt.Errorf("could not find module root (go.mod) starting from %s", wd)
	}

	// Create temp directory for binary
	tmpDir, err := os.MkdirTemp("", "stackit-test-binary-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "stackit")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/stackit")
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tmpDir) // Ignore cleanup errors
		return "", fmt.Errorf("failed to build: %s: %w", string(output), err)
	}

	// Make it executable
	//nolint:gosec // 0755 is correct for an executable binary
	if err := os.Chmod(binaryPath, 0755); err != nil {
		_ = os.RemoveAll(tmpDir) // Ignore cleanup errors
		return "", fmt.Errorf("failed to chmod: %w", err)
	}

	return binaryPath, nil
}

// findModuleRoot walks up the directory tree from startDir to find the module root
// (directory containing go.mod file).
func findModuleRoot(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root of filesystem
			break
		}
		dir = parent
	}
	return ""
}
