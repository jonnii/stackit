// Package integration provides integration tests for stackit CLI commands.
package integration

import (
	"testing"

	"stackit.dev/stackit/internal/testhelper"
)

// getStackitBinary returns the path to the pre-built stackit binary.
func getStackitBinary(t *testing.T) string {
	t.Helper()
	binaryPath := testhelper.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelper.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}
	return binaryPath
}
