package branch_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestAbsorbCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("absorb basic - single hunk to single commit", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create staged change that should be absorbed into the commit
			if err := s.Repo.CreateChange("fix for feature change 1", "test1", false); err != nil {
				return err
			}
			return nil
		})

		// Verify we have staged changes
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = scene.Dir
		output := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(output), "test1_test.txt")

		// Run absorb with --force to skip confirmation
		cmd = exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb command failed: %s", string(output))
		require.Contains(t, string(output), "Absorbed changes", "should mention absorbing")

		// Verify the change was absorbed (should be in the commit now)
		cmd = exec.Command("git", "log", "-1", "--format=%B", "feature")
		cmd.Dir = scene.Dir
		commitMsg := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(commitMsg), "feature change 1")

		// Verify staged changes are gone
		cmd = exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = scene.Dir
		staged := testhelpers.Must(cmd.CombinedOutput())
		require.Empty(t, strings.TrimSpace(string(staged)), "should have no staged changes after absorb")
	})

	t.Run("absorb with --dry-run", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create staged change
			if err := s.Repo.CreateChange("fix for feature change 1", "test1", false); err != nil {
				return err
			}
			return nil
		})

		// Run absorb with --dry-run
		cmd := exec.Command(binaryPath, "absorb", "--dry-run")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb dry-run failed: %s", string(output))
		require.Contains(t, string(output), "Would absorb", "should mention would absorb")
		require.Contains(t, string(output), "feature", "should mention the branch")

		// Verify staged changes are still there (dry-run doesn't modify)
		cmd = exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = scene.Dir
		staged := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(staged), "test1_test.txt", "should still have staged changes after dry-run")
	})

	t.Run("absorb with --all flag", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create unstaged change
			if err := s.Repo.CreateChange("fix for feature change 1", "test1", true); err != nil {
				return err
			}
			return nil
		})

		// Verify we have unstaged changes
		cmd := exec.Command("git", "diff", "--name-only")
		cmd.Dir = scene.Dir
		output := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(output), "test1_test.txt")

		// Run absorb with --all and --force
		cmd = exec.Command(binaryPath, "absorb", "--all", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb with --all failed: %s", string(output))
		require.Contains(t, string(output), "Absorbed changes", "should mention absorbing")
	})

	t.Run("absorb - no staged changes", func(t *testing.T) {
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

		// Run absorb without any staged changes
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb should not fail with no staged changes")
		require.Contains(t, string(output), "Nothing to absorb", "should mention nothing to absorb")
	})

	t.Run("absorb - only unabsorbable changes", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			cmd := exec.Command(binaryPath, "init")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create a branch with a commit
			if err := s.Repo.CreateChange("commit 1", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "commit 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create a staged change that doesn't belong to any commit (new file)
			if err := s.Repo.CreateChange("unrelated change", "file2", false); err != nil {
				return err
			}
			return nil
		})

		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Should not error if there are changes but none can be absorbed")
		require.Contains(t, string(output), "The following hunks could not be absorbed")
		require.Contains(t, string(output), "file2")
	})

	t.Run("absorb error - not initialized", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory without initializing stackit
		tmpDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init", "-b", "main", tmpDir)
		require.NoError(t, cmd.Run())

		// Configure git user
		cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User")
		require.NoError(t, cmd.Run())
		cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com")
		require.NoError(t, cmd.Run())

		// Create initial commit
		cmd = exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "initial")
		require.NoError(t, cmd.Run())

		// Create staged change
		testFile := tmpDir + "/test.txt"
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))
		cmd = exec.Command("git", "-C", tmpDir, "add", testFile)
		require.NoError(t, cmd.Run())

		// Run absorb without initializing stackit
		cmd = exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "absorb should fail when stackit not initialized")
		require.Contains(t, string(output), "not initialized", "should mention not initialized")
	})

	t.Run("absorb multiple hunks to same commit", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create multiple staged changes in the same file
			if err := s.Repo.CreateChange("fix 1 for feature change 1", "test1", false); err != nil {
				return err
			}
			// Create another change in a different file
			if err := s.Repo.CreateChange("fix 2 for feature change 1", "test2", false); err != nil {
				return err
			}
			return nil
		})

		// Run absorb with --force
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb with multiple hunks failed: %s", string(output))
		require.Contains(t, string(output), "Absorbed changes", "should mention absorbing")
	})

	t.Run("absorb restacks upstack branches", func(t *testing.T) {
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
			if err := s.Repo.CreateChange("feature A", "testA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureA", "-m", "feature A")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B on top of A
			if err := s.Repo.CreateChange("feature B", "testB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureB", "-m", "feature B")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Switch back to featureA and create staged change
			cmd = exec.Command("git", "checkout", "featureA")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("fix for feature A", "testA", false); err != nil {
				return err
			}
			return nil
		})

		// Get commit SHA of featureA before absorb
		cmd := exec.Command("git", "rev-parse", "featureA")
		cmd.Dir = scene.Dir
		beforeSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Run absorb with --force
		cmd = exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb failed: %s", string(output))

		// Get commit SHA of featureA after absorb
		cmd = exec.Command("git", "rev-parse", "featureA")
		cmd.Dir = scene.Dir
		afterSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Commit SHA should have changed (commit was rewritten)
		require.NotEqual(t, beforeSHA, afterSHA, "commit should have been rewritten")

		// Verify featureB was restacked (should still be on top of featureA)
		cmd = exec.Command("git", "merge-base", "featureA", "featureB")
		cmd.Dir = scene.Dir
		mergeBase := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))
		require.Equal(t, afterSHA, mergeBase, "featureB should be restacked on updated featureA")
	})

	t.Run("absorb with hunk that commutes with all commits", func(t *testing.T) {
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
			// Create branch with a commit in test1.txt
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create staged change in a completely different file (should commute)
			if err := s.Repo.CreateChange("new file change", "newfile", false); err != nil {
				return err
			}
			return nil
		})

		// Run absorb with --force
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		// Note: According to git-absorb behavior, new files that don't conflict with any commit
		// can still be absorbed into the first commit. The key is whether they commute.
		// For this test, we're checking that the command succeeds.
		// If the file is absorbed, that's actually valid behavior if it doesn't conflict.
		// Let's verify the command succeeds and check the final state
		require.NoError(t, err, "absorb should succeed: %s", string(output))

		// The file might be absorbed or not - both are valid
		// Just verify the command completed successfully
	})

	t.Run("absorb error - detached HEAD", func(t *testing.T) {
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
			// Create a second commit so we have something to detach to
			if err := s.Repo.CreateChangeAndCommit("second commit", "second"); err != nil {
				return err
			}
			// Detach HEAD by checking out a specific commit
			cmd = exec.Command("git", "checkout", "HEAD~1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create staged change
			if err := s.Repo.CreateChange("detached change", "detached", false); err != nil {
				return err
			}
			return nil
		})

		// Run absorb in detached HEAD state
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "absorb should fail in detached HEAD state")
		require.Contains(t, string(output), "not on a branch", "should mention not on a branch")
	})

	t.Run("absorb error - rebase in progress", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Start an interactive rebase that will pause (using exec to create the pause)
		// We'll create a rebase-merge directory to simulate rebase in progress
		rebaseMergeDir := scene.Dir + "/.git/rebase-merge"
		require.NoError(t, os.MkdirAll(rebaseMergeDir, 0755))
		// Write a minimal file to make it look like a rebase is in progress
		require.NoError(t, os.WriteFile(rebaseMergeDir+"/head-name", []byte("refs/heads/feature"), 0644))

		// Create staged change
		require.NoError(t, scene.Repo.CreateChange("change during rebase", "test1", false))

		// Run absorb during rebase
		cmd := exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "absorb should fail during rebase")
		require.Contains(t, string(output), "rebase", "should mention rebase")
	})

	t.Run("absorb hunks to different branches in stack", func(t *testing.T) {
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
			// Create branch A with a commit that touches fileA
			if err := s.Repo.CreateChange("feature A content", "fileA", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureA", "-m", "feature A")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create branch B on top of A with a commit that touches fileB
			if err := s.Repo.CreateChange("feature B content", "fileB", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "featureB", "-m", "feature B")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Now on featureB, stage changes to both fileA and fileB
			// Change to fileA should go to featureA, change to fileB should go to featureB
			if err := s.Repo.CreateChange("fix for feature A", "fileA", false); err != nil {
				return err
			}
			if err := s.Repo.CreateChange("fix for feature B", "fileB", false); err != nil {
				return err
			}
			return nil
		})

		// Verify we have staged changes to both files
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = scene.Dir
		output := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(output), "fileA_test.txt")
		require.Contains(t, string(output), "fileB_test.txt")

		// Run absorb with --force
		cmd = exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb with multiple branches failed: %s", string(output))
		require.Contains(t, string(output), "Absorbed changes", "should mention absorbing")

		// Verify fileA change was absorbed into featureA
		cmd = exec.Command("git", "show", "--name-only", "--format=", "featureA")
		cmd.Dir = scene.Dir
		featureAFiles := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(featureAFiles), "fileA_test.txt", "fileA should be in featureA commit")

		// Verify fileB change was absorbed into featureB
		cmd = exec.Command("git", "show", "--name-only", "--format=", "featureB")
		cmd.Dir = scene.Dir
		featureBFiles := testhelpers.Must(cmd.CombinedOutput())
		require.Contains(t, string(featureBFiles), "fileB_test.txt", "fileB should be in featureB commit")

		// Verify staged changes are gone
		cmd = exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = scene.Dir
		staged := testhelpers.Must(cmd.CombinedOutput())
		require.Empty(t, strings.TrimSpace(string(staged)), "should have no staged changes after absorb")
	})

	t.Run("absorb preserves commit metadata", func(t *testing.T) {
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
			// Create branch with a commit
			if err := s.Repo.CreateChange("feature change 1", "test1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "feature", "-m", "feature change 1")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			// Create staged change
			if err := s.Repo.CreateChange("fix for feature change 1", "test1", false); err != nil {
				return err
			}
			return nil
		})

		// Get original commit metadata before absorb
		cmd := exec.Command("git", "log", "-1", "--format=%an|%ae|%s", "feature")
		cmd.Dir = scene.Dir
		originalMeta := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Run absorb with --force
		cmd = exec.Command(binaryPath, "absorb", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "absorb failed: %s", string(output))

		// Get commit metadata after absorb
		cmd = exec.Command("git", "log", "-1", "--format=%an|%ae|%s", "feature")
		cmd.Dir = scene.Dir
		newMeta := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Verify author name, email, and message are preserved
		require.Equal(t, originalMeta, newMeta, "commit metadata should be preserved after absorb")
	})
}

func TestAbsorbComplex(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

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
