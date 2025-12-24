package stack_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestMoveCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("moves branch downstack with explicit flags", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch2 on top of branch1
			if err := s.Repo.CreateChange("branch2 change", "file2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "branch2 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Verify initial state: branch2 should have branch1 as parent
		cmd := exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branch1", "branch2 should initially have branch1 as parent")

		// Move branch2 from branch1 to main
		cmd = exec.Command(binaryPath, "move", "--source", "branch2", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "move command failed: %s", string(output))
		require.Contains(t, string(output), "Moved", "should mention moving")
		require.Contains(t, string(output), "branch2", "should mention branch2")
		require.Contains(t, string(output), "main", "should mention main")

		// Verify new parent relationship
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "main", "branch2 should now have main as parent")
		require.NotContains(t, string(output), "branch1", "branch2 should no longer have branch1 as parent")
	})

	t.Run("moves current branch when source not specified", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch2 on top of branch1
			if err := s.Repo.CreateChange("branch2 change", "file2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "branch2 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Switch to branch2
		err := scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)

		// Move current branch (branch2) to main without specifying source
		cmd := exec.Command(binaryPath, "move", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "move command failed: %s", string(output))
		require.Contains(t, string(output), "Moved", "should mention moving")

		// Verify new parent relationship
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "main", "branch2 should now have main as parent")
	})

	t.Run("moves branch upstack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branchA
			if err := s.Repo.CreateChange("branchA change", "fileA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchA", "-m", "branchA change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branchB (sibling of branchA)
			err := s.Repo.CheckoutBranch("main")
			if err != nil {
				return err
			}
			if err := s.Repo.CreateChange("branchB change", "fileB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchB", "-m", "branchB change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Move branchA from main to branchB (upstack - moving to a sibling branch)
		cmd := exec.Command(binaryPath, "move", "--source", "branchA", "--onto", "branchB")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "move command failed: %s", string(output))
		require.Contains(t, string(output), "Moved", "should mention moving")

		// Verify new parent relationship
		cmd = exec.Command(binaryPath, "checkout", "branchA")
		cmd.Dir = scene.Dir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branchB", "branchA should now have branchB as parent")
	})

	t.Run("restacks descendants after move", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch2 on top of branch1
			if err := s.Repo.CreateChange("branch2 change", "file2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "branch2 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch3 on top of branch2
			if err := s.Repo.CreateChange("branch3 change", "file3", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch3", "-m", "branch3 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Make a change to main to force restacking
		err := scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("new main change", "main")
		require.NoError(t, err)

		// Move branch1 to main (which now has new commits)
		cmd := exec.Command(binaryPath, "move", "--source", "branch1", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "move command failed: %s", string(output))
		require.Contains(t, string(output), "Moved", "should mention moving")
		require.Contains(t, string(output), "Restacked", "should mention restacking")

		// Verify all branches are properly restacked (check that they're fixed)
		cmd = exec.Command(binaryPath, "checkout", "branch1")
		cmd.Dir = scene.Dir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "info")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info command failed: %s", string(output))
		// Should not mention "fallen behind" which indicates branches are properly restacked
		require.NotContains(t, string(output), "fallen behind", "branches should be properly restacked")
	})

	t.Run("fails when trying to move trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Try to move trunk
		cmd := exec.Command(binaryPath, "move", "--source", "main", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when trying to move trunk")
		require.Contains(t, string(output), "cannot move trunk branch", "should mention trunk cannot be moved")
	})

	t.Run("fails when trying to move onto descendant", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch2 on top of branch1
			if err := s.Repo.CreateChange("branch2 change", "file2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "branch2 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Try to move branch1 onto branch2 (which is a descendant of branch1)
		cmd := exec.Command(binaryPath, "move", "--source", "branch1", "--onto", "branch2")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when trying to move onto descendant")
		require.Contains(t, string(output), "cannot move", "should mention cannot move")
		require.Contains(t, string(output), "descendant", "should mention descendant")
	})

	t.Run("fails when trying to move onto itself", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Try to move branch1 onto itself
		cmd := exec.Command(binaryPath, "move", "--source", "branch1", "--onto", "branch1")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when trying to move onto itself")
		require.Contains(t, string(output), "cannot move branch onto itself", "should mention cannot move onto itself")
	})

	t.Run("fails when source branch is not tracked", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create untracked branch
			err := s.Repo.CreateAndCheckoutBranch("untracked")
			if err != nil {
				return err
			}
			err = s.Repo.CreateChangeAndCommit("untracked change", "u")
			if err != nil {
				return err
			}
			return nil
		})

		// Try to move untracked branch
		cmd := exec.Command(binaryPath, "move", "--source", "untracked", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when source is not tracked")
		require.Contains(t, string(output), "not tracked", "should mention not tracked")
	})

	t.Run("fails when onto branch does not exist", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Try to move branch1 onto nonexistent branch
		cmd := exec.Command(binaryPath, "move", "--source", "branch1", "--onto", "nonexistent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when onto branch does not exist")
		require.Contains(t, string(output), "does not exist", "should mention branch does not exist")
	})

	t.Run("fails when not on branch and no source specified", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Detach HEAD
		err := scene.Repo.RunGitCommand("checkout", "HEAD")
		require.NoError(t, err)

		// Try to move without specifying source
		cmd := exec.Command(binaryPath, "move", "--onto", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "move should fail when not on branch and no source specified")
		// The error should be either "not on a branch" or "cannot move trunk branch"
		// depending on what CurrentBranch() returns
		outputStr := string(output)
		require.True(t,
			strings.Contains(outputStr, "not on a branch") || strings.Contains(outputStr, "cannot move trunk branch"),
			"should mention not on a branch or trunk: %s", outputStr)
	})

	t.Run("moves branch across different stack trees", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create stack A: branchA1 -> branchA2
			if err := s.Repo.CreateChange("branchA1 change", "fileA1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchA1", "-m", "branchA1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("branchA2 change", "fileA2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchA2", "-m", "branchA2 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create stack B: branchB1 -> branchB2
			err := s.Repo.CheckoutBranch("main")
			if err != nil {
				return err
			}
			if err := s.Repo.CreateChange("branchB1 change", "fileB1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchB1", "-m", "branchB1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("branchB2 change", "fileB2", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchB2", "-m", "branchB2 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Move branchA2 from branchA1 to branchB1 (across stacks)
		cmd := exec.Command(binaryPath, "move", "--source", "branchA2", "--onto", "branchB1")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "move command failed: %s", string(output))
		require.Contains(t, string(output), "Moved", "should mention moving")

		// Verify new parent relationship
		cmd = exec.Command(binaryPath, "checkout", "branchA2")
		cmd.Dir = scene.Dir
		require.NoError(t, cmd.Run())

		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branchB1", "branchA2 should now have branchB1 as parent")
	})

	t.Run("interactive selection fails in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			return cmd.Run()
		})

		// Disable interactive mode
		cmd := exec.Command(binaryPath, "move")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_TEST_NO_INTERACTIVE=1")
		output, err := cmd.CombinedOutput()
		// Should fail because interactive selection is disabled or onto not specified
		require.Error(t, err, "move should fail in non-interactive mode")
		// The error might be about interactive prompts being disabled or onto not being specified
		outputStr := string(output)
		require.True(t,
			strings.Contains(outputStr, "interactive") || strings.Contains(outputStr, "onto branch must be specified") || strings.Contains(outputStr, "not on a branch"),
			"should mention interactive, onto requirement, or not on branch: %s", outputStr)
	})
}
