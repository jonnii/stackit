package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

func TestInfoCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("basic info display on current branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with commit
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info command
		cmd := exec.Command(binaryPath, "info")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		require.Contains(t, outputStr, "feature", "should contain branch name")
		require.Contains(t, outputStr, "(current)", "should indicate current branch")
		require.Contains(t, outputStr, "feature change", "should contain commit message")
	})

	t.Run("info for specified branch argument", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch A
			if err := s.Repo.CreateChange("a change", "a", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "a", "-m", "a change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B
			if err := s.Repo.CreateChange("b change", "b", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "b", "-m", "b change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Switch to branch b
		require.NoError(t, scene.Repo.CheckoutBranch("b"))

		// Run info for branch a
		cmd := exec.Command(binaryPath, "info", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		require.Contains(t, outputStr, "a", "should contain branch name")
		require.Contains(t, outputStr, "a change", "should contain commit message")
		require.NotContains(t, outputStr, "(current)", "should not indicate current branch")
	})

	t.Run("info shows parent branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch A
			if err := s.Repo.CreateChange("a change", "a", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "a", "-m", "a change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B on top of A
			if err := s.Repo.CreateChange("b change", "b", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "b", "-m", "b change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info for branch b
		cmd := exec.Command(binaryPath, "info", "b")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		require.Contains(t, outputStr, "Parent", "should show parent section")
		require.Contains(t, outputStr, "a", "should show parent branch name")
	})

	t.Run("info shows children branches", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch A
			if err := s.Repo.CreateChange("a change", "a", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "a", "-m", "a change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B on top of A
			if err := s.Repo.CreateChange("b change", "b", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "b", "-m", "b change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info for branch a
		cmd := exec.Command(binaryPath, "info", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		require.Contains(t, outputStr, "Children", "should show children section")
		require.Contains(t, outputStr, "b", "should show child branch name")
	})

	t.Run("info with --diff flag shows diff", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with change
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info with --diff
		cmd := exec.Command(binaryPath, "info", "--diff")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		// Should contain diff markers
		require.True(t, strings.Contains(outputStr, "+++") || strings.Contains(outputStr, "---") || strings.Contains(outputStr, "@@"),
			"should contain diff output, got: %s", outputStr)
	})

	t.Run("info with --patch flag shows patches", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with change
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info with --patch
		cmd := exec.Command(binaryPath, "info", "--patch")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		// Should contain patch markers
		require.True(t, strings.Contains(outputStr, "+++") || strings.Contains(outputStr, "---") || strings.Contains(outputStr, "@@"),
			"should contain patch output, got: %s", outputStr)
	})

	t.Run("info with --stat flag shows diffstat", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with change
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info with --stat (implies --diff)
		cmd := exec.Command(binaryPath, "info", "--stat")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		// Should contain stat output (file names and change counts)
		require.True(t, strings.Contains(outputStr, "test") || strings.Contains(outputStr, "|"),
			"should contain diffstat output, got: %s", outputStr)
	})

	t.Run("info with --stat --diff shows diffstat", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with change
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info with --stat --diff
		cmd := exec.Command(binaryPath, "info", "--stat", "--diff")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		// Should contain stat output
		require.True(t, strings.Contains(outputStr, "test") || strings.Contains(outputStr, "|"),
			"should contain diffstat output, got: %s", outputStr)
	})

	t.Run("info with --stat --patch shows stat per commit", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch with change
			if err := s.Repo.CreateChange("feature change", "test", false); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Run info with --stat --patch
		cmd := exec.Command(binaryPath, "info", "--stat", "--patch")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		// Should contain stat output
		require.True(t, strings.Contains(outputStr, "test") || strings.Contains(outputStr, "|"),
			"should contain diffstat output, got: %s", outputStr)
	})

	t.Run("info errors on non-existent branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Run info for non-existent branch
		cmd := exec.Command(binaryPath, "info", "nonexistent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "info should fail for non-existent branch")
		require.Contains(t, string(output), "does not exist", "should mention branch does not exist")
	})

	t.Run("info works on trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Run info on trunk (main)
		cmd := exec.Command(binaryPath, "info", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "info command failed: %s", string(output))
		outputStr := string(output)
		require.Contains(t, outputStr, "main", "should contain trunk branch name")
	})

	t.Run("info errors when not on branch and no branch specified", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Detach HEAD
		require.NoError(t, scene.Repo.RunGitCommand("checkout", "HEAD~0"))

		// Run info without branch argument
		cmd := exec.Command(binaryPath, "info")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "info should fail when not on branch")
		require.Contains(t, string(output), "not on a branch", "should mention not on branch")
	})

	t.Run("info errors when stackit not initialized", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory without stackit initialization
		tmpDir := t.TempDir()
		cmd := exec.Command("git", "init", tmpDir, "-b", "main")
		require.NoError(t, cmd.Run())

		// Create a commit
		cmd = exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "initial")
		require.NoError(t, cmd.Run())

		// Try to run info without initializing stackit
		cmd = exec.Command(binaryPath, "info")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "info should fail when stackit not initialized")
		require.Contains(t, string(output), "not initialized", "should mention not initialized")
	})
}
