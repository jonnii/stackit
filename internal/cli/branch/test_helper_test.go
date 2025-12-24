package branch_test

import (
	"testing"

	"stackit.dev/stackit/internal/testhelper"
)

// getStackitBinary returns the path to the pre-built stackit binary.
// This is a wrapper around the shared binary path from the parent cli_test package.
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
