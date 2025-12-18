package cli_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestAbsorbComplex(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("absorb restacks multiple child branches in branching stack", func(t *testing.T) {
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
			// Create branch A
			if err := s.Repo.CreateChange("feature A content", "fileA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureA", "-m", "feature A")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B on top of A
			if err := s.Repo.CreateChange("feature B content", "fileB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureB", "-m", "feature B")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Go back to A and create branch C on top of A
			cmd = exec.Command("git", "checkout", "featureA")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("feature C content", "fileC", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureC", "-m", "feature C")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}

			// Now we are on featureC. featureA has two children: featureB and featureC.
			// Go back to featureA and stage a change.
			cmd = exec.Command("git", "checkout", "featureA")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("fix for feature A", "fileA", false); err != nil {
				return err
			}
			return nil
		})

		// Run absorb
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "absorb failed: %s", string(output))

		// Get new SHA for featureA
		cmd = exec.Command("git", "rev-parse", "featureA")
		cmd.Dir = scene.Dir
		afterSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Verify featureB was restacked
		cmd = exec.Command("git", "merge-base", "featureA", "featureB")
		cmd.Dir = scene.Dir
		mergeBaseB := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))
		require.Equal(t, afterSHA, mergeBaseB, "featureB should be restacked on updated featureA")

		// Verify featureC was restacked
		cmd = exec.Command("git", "merge-base", "featureA", "featureC")
		cmd.Dir = scene.Dir
		mergeBaseC := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))
		require.Equal(t, afterSHA, mergeBaseC, "featureC should be restacked on updated featureA")
	})

	t.Run("absorb hunks into different commits across different files", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Initialize stackit
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Branch 1: Modify fileA
			if err := s.Repo.CreateChange("fileA content", "fileA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "add fileA")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Branch 2: Modify fileB
			if err := s.Repo.CreateChange("fileB content", "fileB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch2", "-m", "add fileB")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Stage changes for both files
			if err := s.Repo.CreateChange("fileA content fix", "fileA", false); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("fileB content fix", "fileB", false); err != nil {
				return err
			}

			return nil
		})

		// Run absorb
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "absorb failed: %s", string(output))

		// Verify branch1 contains fix for fileA
		cmd = exec.Command("git", "show", "branch1:fileA_test.txt")
		cmd.Dir = scene.Dir
		content := string(testhelpers.Must(cmd.CombinedOutput()))
		require.Contains(t, content, "fileA content fix")

		// Verify branch2 contains fix for fileB
		cmd = exec.Command("git", "show", "branch2:fileB_test.txt")
		cmd.Dir = scene.Dir
		content = string(testhelpers.Must(cmd.CombinedOutput()))
		require.Contains(t, content, "fileB content fix")
	})

	t.Run("absorb failure during restack preserves work and reports error", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create branch A
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			err := os.WriteFile(s.Dir+"/conflict.txt", []byte("line 1\nline 2\nline 3\n"), 0644)
			require.NoError(t, err)
			cmd = exec.Command("git", "add", "conflict.txt")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())
			cmd = exec.Command(binaryPath, "create", "branchA", "-m", "add conflict.txt", "--all")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Branch B modifies line 2
			err = os.WriteFile(s.Dir+"/conflict.txt", []byte("line 1\nline 2 B\nline 3\n"), 0644)
			require.NoError(t, err)
			cmd = exec.Command(binaryPath, "create", "branchB", "-m", "modify line 2", "--all")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Go back to branchA and absorb a change that modifies line 2, which will cause a conflict when restacking branchB
			cmd = exec.Command("git", "checkout", "branchA")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Important: change line 1 so it absorbs into branchA, but change line 2 to something else to cause conflict with branchB
			err = os.WriteFile(s.Dir+"/conflict.txt", []byte("line 1 modified\nline 2 modified in A\nline 3\n"), 0644)
			require.NoError(t, err)
			cmd = exec.Command("git", "add", "conflict.txt")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			return nil
		})

		// Run absorb. It should successfully absorb into branchA, but then fail during restack of branchB
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "absorb should fail during restack. Output: %s", string(output))
		require.Contains(t, string(output), "failed to restack", "should report restack failure")

		// In case of restack conflict, stackit stays in rebase mode (detached HEAD)
		rebaseDir := scene.Dir + "/.git/rebase-merge"
		if _, err := os.Stat(rebaseDir); os.IsNotExist(err) {
			rebaseDir = scene.Dir + "/.git/rebase-apply"
		}
		_, err = os.Stat(rebaseDir)
		require.NoError(t, err, "should be in middle of rebase")

		// Clean up for next tests
		cmd = exec.Command("git", "rebase", "--abort")
		cmd.Dir = scene.Dir
		_ = cmd.Run()
	})

	t.Run("absorb into ancestor restacks all intermediate branches to tip", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Stack: main -> branchA -> branchB (current)
			if err := s.Repo.CreateChange("content A", "fileA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchA", "-m", "add fileA")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			if err := s.Repo.CreateChange("content B", "fileB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branchB", "-m", "add fileB")
			cmd.Dir = s.Dir
			require.NoError(t, cmd.Run())

			// Stay on branchB. If we absorb into branchA, branchB MUST be restacked.
			if err := s.Repo.CreateChange("content A fix", "fileA", false); err != nil {
				return err
			}

			return nil
		})

		// Run absorb
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "absorb failed: %s", string(output))

		// Verify branchA was updated
		cmd = exec.Command("git", "rev-parse", "branchA")
		cmd.Dir = scene.Dir
		afterSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Verify branchB was restacked onto new branchA
		cmd = exec.Command("git", "merge-base", "branchA", "branchB")
		cmd.Dir = scene.Dir
		mergeBase := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))
		require.Equal(t, afterSHA, mergeBase, "branchB should be restacked on updated branchA")
	})
}
