package testhelpers

import (
	"os"
	"path/filepath"
	"testing"

	"stackit.dev/stackit/internal/git"
)

// Scene represents a test scene with a temporary directory and Git repository.
// This is the Go equivalent of the TypeScript AbstractScene.
type Scene struct {
	Dir    string
	Repo   *GitRepo
	oldDir string
}

// SceneSetup is a function type for setting up a scene.
type SceneSetup func(*Scene) error

// NewScene creates a new test scene with a temporary directory and Git repository.
// It automatically handles cleanup using t.Cleanup().
// NOTE: This function uses os.Chdir() and is NOT safe for parallel tests.
// Use NewSceneParallel for tests that can run in parallel.
func NewScene(t *testing.T, setup SceneSetup) *Scene {
	// Reset the default git repository to ensure this test gets a fresh one.
	// This is necessary because the git package uses a package-level global
	// for the repository, which would otherwise persist across tests.
	git.ResetDefaultRepo()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "stackit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Initialize Git repository
	repo, err := NewGitRepo(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create Git repo: %v", err)
	}

	scene := &Scene{
		Dir:    tmpDir,
		Repo:   repo,
		oldDir: oldDir,
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Write default config files
	if err := scene.writeDefaultConfigs(); err != nil {
		os.Chdir(oldDir)
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write config files: %v", err)
	}

	// Run custom setup if provided
	if setup != nil {
		if err := setup(scene); err != nil {
			os.Chdir(oldDir)
			os.RemoveAll(tmpDir)
			t.Fatalf("Setup failed: %v", err)
		}
	}

	// Register cleanup
	t.Cleanup(func() {
		os.Chdir(oldDir)
		if os.Getenv("DEBUG") == "" {
			os.RemoveAll(tmpDir)
		}
	})

	return scene
}

// NewSceneParallel creates a new test scene that is safe for parallel tests.
// Unlike NewScene, this does NOT change the working directory.
// Tests using this must ensure all git operations use explicit directory paths
// (e.g., via scene.Repo methods or cmd.Dir = scene.Dir).
func NewSceneParallel(t *testing.T, setup SceneSetup) *Scene {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "stackit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize Git repository
	repo, err := NewGitRepo(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create Git repo: %v", err)
	}

	scene := &Scene{
		Dir:  tmpDir,
		Repo: repo,
	}

	// Write default config files
	if err := scene.writeDefaultConfigs(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write config files: %v", err)
	}

	// Run custom setup if provided
	if setup != nil {
		if err := setup(scene); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Setup failed: %v", err)
		}
	}

	// Register cleanup
	t.Cleanup(func() {
		if os.Getenv("DEBUG") == "" {
			os.RemoveAll(tmpDir)
		}
	})

	return scene
}

// writeDefaultConfigs writes the default Stackit configuration files.
func (s *Scene) writeDefaultConfigs() error {
	// Write repo config (JSON format, matching cuteString output)
	repoConfigPath := filepath.Join(s.Dir, ".git", ".stackit_config")
	repoConfig := `{
  "trunk": "main",
  "isGithubIntegrationEnabled": false
}
`
	if err := os.WriteFile(repoConfigPath, []byte(repoConfig), 0644); err != nil {
		return err
	}

	// Write user config (JSON format)
	userConfigPath := filepath.Join(s.Dir, ".git", ".stackit_user_config")
	userConfig := `{
  "tips": false
}
`
	if err := os.WriteFile(userConfigPath, []byte(userConfig), 0644); err != nil {
		return err
	}

	// Set environment variable for user config path
	os.Setenv("STACKIT_USER_CONFIG_PATH", userConfigPath)
	os.Setenv("STACKIT_PROFILE", "")

	return nil
}

// BasicSceneSetup is a setup function that creates a basic scene with a single commit.
func BasicSceneSetup(scene *Scene) error {
	return scene.Repo.CreateChangeAndCommit("1", "1")
}
