package cli_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestScopeCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("scope set fails on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Try to set scope on trunk (main)
		cmd = exec.Command(binaryPath, "scope", "PROJ-123")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "scope set should fail on trunk")
		require.Contains(t, string(output), "cannot set scope on trunk")
	})

	t.Run("scope unset fails on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Try to unset scope on trunk (main)
		cmd = exec.Command(binaryPath, "scope", "--unset")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "scope unset should fail on trunk")
		require.Contains(t, string(output), "cannot unset scope on trunk")
	})

	t.Run("scope show fails on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Try to show scope on trunk (main)
		cmd = exec.Command(binaryPath, "scope", "--show")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "scope show should fail on trunk")
		require.Contains(t, string(output), "not on a branch")
	})
}
