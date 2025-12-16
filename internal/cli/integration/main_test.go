package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// sharedBinaryPath holds the path to the pre-built stackit binary.
// It's built once in TestMain and reused by all tests.
var sharedBinaryPath string

func TestMain(m *testing.M) {
	// Build the binary once before running any tests
	binaryPath, cleanup, err := buildBinaryOnce()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build stackit binary: %v\n", err)
		os.Exit(1)
	}
	sharedBinaryPath = binaryPath

	// Run all tests
	code := m.Run()

	// Cleanup
	cleanup()
	os.Exit(code)
}

func buildBinaryOnce() (string, func(), error) {
	// Get the module root (go up from internal/cli/integration to stackit root)
	wd, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	moduleRoot := filepath.Join(wd, "..", "..", "..")

	// Create temp directory for binary
	tmpDir, err := os.MkdirTemp("", "stackit-integration-test-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "stackit")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/stackit")
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("failed to build: %s: %w", string(output), err)
	}

	// Make it executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("failed to chmod: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return binaryPath, cleanup, nil
}

// getStackitBinary returns the path to the pre-built stackit binary.
func getStackitBinary(t *testing.T) string {
	t.Helper()
	if sharedBinaryPath == "" {
		t.Fatal("stackit binary not built - TestMain should have built it")
	}
	return sharedBinaryPath
}
