package cli_test

import (
	"testing"

	"stackit.dev/stackit/testhelpers"
)

func TestMain(m *testing.M) {
	testhelpers.TestMain(m, nil)
}

// getStackitBinary returns the path to the pre-built stackit binary.
func getStackitBinary(t *testing.T) string {
	t.Helper()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}
	return binaryPath
}
