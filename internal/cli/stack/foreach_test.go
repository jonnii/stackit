package stack_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestForeachCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("basic foreach on single branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		cmd := exec.Command(binaryPath, "foreach", "touch", "touched")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "foreach command failed: %s", string(output))
		require.Contains(t, string(output), "Running on branch branch1", "should mention branch1")
		require.FileExists(t, scene.Dir+"/touched", "file should be created")
	})

	t.Run("foreach on multiple branches (upstack)", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2 -> branch3
			for _, b := range []string{"branch1", "branch2", "branch3"} {
				cmd := exec.Command(binaryPath, "create", b, "-m", b+" change")
				cmd.Dir = s.Dir
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			// Checkout branch1 to test upstack from there
			return s.Repo.CheckoutBranch("branch1")
		})

		// The command doesn't automatically export ST_BRANCH yet.
		// Let's use a simple command first.
		cmd := exec.Command(binaryPath, "foreach", "echo", "hello")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "foreach command failed: %s", string(output))
		require.Contains(t, string(output), "Running on branch branch1", "should mention branch1")
		require.Contains(t, string(output), "Running on branch branch2", "should mention branch2")
		require.Contains(t, string(output), "Running on branch branch3", "should mention branch3")
	})

	t.Run("foreach --stack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2 -> branch3
			for _, b := range []string{"branch1", "branch2", "branch3"} {
				cmd := exec.Command(binaryPath, "create", b, "-m", b+" change")
				cmd.Dir = s.Dir
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			// Checkout branch2
			return s.Repo.CheckoutBranch("branch2")
		})

		// --stack should run on branch1, branch2, branch3
		cmd := exec.Command(binaryPath, "foreach", "--stack", "echo", "hello")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "foreach command failed: %s", string(output))
		require.Contains(t, string(output), "Running on branch branch1")
		require.Contains(t, string(output), "Running on branch branch2")
		require.Contains(t, string(output), "Running on branch branch3")
	})

	t.Run("foreach --downstack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2 -> branch3
			for _, b := range []string{"branch1", "branch2", "branch3"} {
				cmd := exec.Command(binaryPath, "create", b, "-m", b+" change")
				cmd.Dir = s.Dir
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			// Checkout branch2
			return s.Repo.CheckoutBranch("branch2")
		})

		// --downstack should run on branch1, branch2
		cmd := exec.Command(binaryPath, "foreach", "--downstack", "echo", "hello")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "foreach command failed: %s", string(output))
		require.Contains(t, string(output), "Running on branch branch1")
		require.Contains(t, string(output), "Running on branch branch2")
		require.NotContains(t, string(output), "Running on branch branch3")
	})

	t.Run("foreach fail-fast", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2
			for _, b := range []string{"branch1", "branch2"} {
				cmd := exec.Command(binaryPath, "create", b, "-m", b+" change")
				cmd.Dir = s.Dir
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		// fail-fast (default) should stop at branch1
		cmd := exec.Command(binaryPath, "foreach", "false")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err)
		require.Contains(t, string(output), "Running on branch branch1")
		require.NotContains(t, string(output), "Running on branch branch2")
	})

	t.Run("foreach --no-fail-fast", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1 -> branch2
			for _, b := range []string{"branch1", "branch2"} {
				cmd := exec.Command(binaryPath, "create", b, "-m", b+" change")
				cmd.Dir = s.Dir
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			return s.Repo.CheckoutBranch("branch1")
		})

		// --no-fail-fast should continue to branch2 even if branch1 fails
		cmd := exec.Command(binaryPath, "foreach", "--no-fail-fast", "false")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err) // Cobra might still return success if it handled the errors internally without returning them to RunE
		// Wait, ForeachAction returns err if opts.FailFast is true.
		// If opts.FailFast is false, it continues and returns nil at the end.

		require.Contains(t, string(output), "Running on branch branch1")
		require.Contains(t, string(output), "Running on branch branch2")
	})

	t.Run("foreach on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// branch1
			cmd := exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Checkout main
			return s.Repo.CheckoutBranch("main")
		})

		// Running on main should skip main and run on descendants if upstack (default)
		cmd := exec.Command(binaryPath, "foreach", "echo", "hello")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "foreach command failed: %s", string(output))
		require.NotContains(t, string(output), "Running on branch main")
		require.Contains(t, string(output), "Running on branch branch1")
	})
}
