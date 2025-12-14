package testhelpers

import (
	"os"
	"path/filepath"
	"testing"
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
func NewScene(t *testing.T, setup SceneSetup) *Scene {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "charcoal-test-*")
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

// writeDefaultConfigs writes the default Graphite configuration files.
func (s *Scene) writeDefaultConfigs() error {
	// Write repo config
	repoConfigPath := filepath.Join(s.Dir, ".git", ".graphite_repo_config")
	repoConfig := `trunk: main
isGithubIntegrationEnabled: false
`
	if err := os.WriteFile(repoConfigPath, []byte(repoConfig), 0644); err != nil {
		return err
	}

	// Write user config
	userConfigPath := filepath.Join(s.Dir, ".git", ".graphite_user_config")
	userConfig := `tips: false
`
	if err := os.WriteFile(userConfigPath, []byte(userConfig), 0644); err != nil {
		return err
	}

	// Set environment variable for user config path
	os.Setenv("GRAPHITE_USER_CONFIG_PATH", userConfigPath)
	os.Setenv("GRAPHITE_PROFILE", "")

	return nil
}

// BasicSceneSetup is a setup function that creates a basic scene with a single commit.
func BasicSceneSetup(scene *Scene) error {
	return scene.Repo.CreateChangeAndCommit("1", "1")
}
