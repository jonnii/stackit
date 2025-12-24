package stack_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestReorderCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("simple reorder: A -> B to B -> A", func(t *testing.T) {
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

		// Create editor script that reorders: branch2, branch1 (reversed)
		editorScript := createEditorScript(t, "branch2\nbranch1\n")

		// Run reorder command
		cmd = exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "EDITOR="+editorScript)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "reorder command failed: %s", string(output))
		require.Contains(t, string(output), "Reordered", "should mention reordering")

		// Verify new parent relationship: branch2 should now have main as parent
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "main", "branch2 should now have main as parent")

		// Checkout branch1 and verify it has branch2 as parent
		cmd = exec.Command("git", "checkout", "branch1")
		cmd.Dir = scene.Dir
		require.NoError(t, cmd.Run())
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branch2", "branch1 should now have branch2 as parent")
	})

	t.Run("reorder with descendants", func(t *testing.T) {
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
			// Create branch3 on top of branch2 (descendant)
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

		// Verify initial state: branch3 should have branch2 as parent
		cmd := exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branch2", "branch3 should initially have branch2 as parent")

		// Create editor script that reorders: branch2, branch1, branch3
		// This reorders branch1 and branch2 (reversed), keeping branch3 at the end
		// After reordering: trunk -> branch2 -> branch1 -> branch3
		// branch3's parent becomes branch1 (the branch before it in the new order)
		editorScript := createEditorScript(t, "branch2\nbranch1\nbranch3\n")

		// Run reorder command
		cmd = exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "EDITOR="+editorScript)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "reorder command failed: %s", string(output))

		// Verify branch3 now has branch1 as parent (new order: branch2 -> branch1 -> branch3)
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Contains(t, string(output), "branch1", "branch3 should now have branch1 as parent after reorder")
	})

	t.Run("error when branch is removed", func(t *testing.T) {
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

		// Create editor script that removes branch1
		editorScript := createEditorScript(t, "branch2\n")

		// Run reorder command - should fail
		cmd := exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "EDITOR="+editorScript)
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "reorder should fail when branch is removed")
		require.Contains(t, string(output), "was removed", "error should mention branch removal")
		require.Contains(t, string(output), "stackit delete", "error should suggest using stackit delete")
	})

	t.Run("no changes when order unchanged", func(t *testing.T) {
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

		// Create editor script with original order (branch1, branch2)
		editorScript := createEditorScript(t, "branch1\nbranch2\n")

		// Run reorder command
		cmd := exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "EDITOR="+editorScript)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "reorder command failed: %s", string(output))
		require.Contains(t, string(output), "unchanged", "should mention order unchanged")
	})

	t.Run("error when less than 2 branches", func(t *testing.T) {
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
			// Create only one branch
			if err := s.Repo.CreateChange("branch1 change", "file1", false); err != nil {
				return err
			}
			cmd = exec.Command(binaryPath, "create", "branch1", "-m", "branch1 change")
			cmd.Dir = s.Dir
			if err := cmd.Run(); err != nil {
				return err
			}
			return nil
		})

		// Run reorder command - should fail
		cmd := exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "reorder should fail with less than 2 branches")
		require.Contains(t, string(output), "at least 2 branches", "error should mention need for 2+ branches")
	})

	t.Run("error when on trunk", func(t *testing.T) {
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
			return nil
		})

		// Run reorder command while on trunk - should fail
		cmd := exec.Command(binaryPath, "reorder")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "reorder should fail when on trunk")
		require.Contains(t, string(output), "cannot reorder trunk", "error should mention trunk")
	})
}

// createEditorScript creates a shell script in a temp directory that writes the given content to the file
// passed as the first argument, simulating an editor
func createEditorScript(t *testing.T, content string) string {
	// Create temp directory outside the repo to avoid git seeing it as uncommitted changes
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "editor.sh")
	// Use a heredoc to safely write content
	scriptContent := "#!/bin/sh\n"
	scriptContent += "cat > \"$1\" << 'EOF'\n"
	scriptContent += content
	scriptContent += "EOF\n"

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Return absolute path for EDITOR environment variable
	absPath, err := filepath.Abs(scriptPath)
	require.NoError(t, err)
	return absPath
}
