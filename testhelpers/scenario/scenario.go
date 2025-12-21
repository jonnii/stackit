// Package scenario provides a high-level test scenario that combines a Scene,
// an Engine, and a runtime Context to provide a terse API for integration tests.
package scenario

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

// Scenario represents a high-level test scenario that combines a Scene,
// an Engine, and a runtime Context to provide a terse API for integration tests.
type Scenario struct {
	T          *testing.T
	Scene      *testhelpers.Scene
	Engine     engine.Engine
	Context    *runtime.Context
	BinaryPath string
}

// NewScenario creates a new Scenario with an optional setup function.
// NOTE: This function is NOT safe for parallel tests as it uses t.Setenv and NewScene.
func NewScenario(t *testing.T, setup testhelpers.SceneSetup) *Scenario {
	t.Helper()

	// Force non-interactive mode for tests
	t.Setenv("STACKIT_NON_INTERACTIVE", "true")

	scene := testhelpers.NewScene(t, setup)
	eng, err := engine.NewEngine(scene.Dir)
	require.NoError(t, err)

	ctx := runtime.NewContext(eng)
	ctx.RepoRoot = scene.Dir

	return &Scenario{
		T:       t,
		Scene:   scene,
		Engine:  eng,
		Context: ctx,
	}
}

// NewScenarioParallel creates a new Scenario that is safe for parallel tests.
// It does NOT set global environment variables or initialize the Go Engine/Context.
// Use this for tests that primarily call the CLI binary.
func NewScenarioParallel(t *testing.T, setup testhelpers.SceneSetup) *Scenario {
	t.Helper()
	scene := testhelpers.NewSceneParallel(t, setup)
	return &Scenario{
		T:     t,
		Scene: scene,
	}
}

// WithInitialCommit creates an initial commit on the main branch.
func (s *Scenario) WithInitialCommit() *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.CreateChangeAndCommit("initial", "init")
	require.NoError(s.T, err)
	return s
}

// WithUncommittedChange creates an uncommitted change in the repository.
func (s *Scenario) WithUncommittedChange(name string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.CreateChange("unstaged content", name, true)
	require.NoError(s.T, err)
	return s
}

// RunGit runs a git command in the scenario's repository.
func (s *Scenario) RunGit(args ...string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.RunGitCommand(args...)
	require.NoError(s.T, err)
	return s
}

// Checkout checks out a branch and rebuilds the engine.
func (s *Scenario) Checkout(branch string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.CheckoutBranch(branch)
	require.NoError(s.T, err)
	return s.Rebuild()
}

// CreateBranch creates and checks out a new branch and rebuilds the engine.
func (s *Scenario) CreateBranch(name string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.CreateAndCheckoutBranch(name)
	require.NoError(s.T, err)
	return s.Rebuild()
}

// Rebuild refreshes the engine's internal state from the Git repository.
func (s *Scenario) Rebuild() *Scenario {
	s.T.Helper()
	git.ResetDefaultRepo()
	err := git.InitDefaultRepo()
	require.NoError(s.T, err)
	if s.Engine != nil {
		err = s.Engine.Rebuild(context.Background(), s.Engine.Trunk())
		require.NoError(s.T, err)
	}
	return s
}

// Commit creates an empty commit with the given message.
func (s *Scenario) Commit(message string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.RunGitCommand("commit", "--allow-empty", "-m", message)
	require.NoError(s.T, err)
	return s
}

// CommitChange creates a file change and commits it.
func (s *Scenario) CommitChange(name, message string) *Scenario {
	s.T.Helper()
	err := s.Scene.Repo.CreateChangeAndCommit(message, name)
	require.NoError(s.T, err)
	return s
}

// TrackBranch tracks a branch with a parent in the engine.
func (s *Scenario) TrackBranch(branch, parent string) *Scenario {
	s.T.Helper()
	err := s.Engine.TrackBranch(context.Background(), branch, parent)
	require.NoError(s.T, err)
	return s
}

// WithStack sets up a branch hierarchy. The map keys are branch names,
// and values are their parent branch names.
// It automatically creates a commit on each branch and tracks it.
func (s *Scenario) WithStack(structure map[string]string) *Scenario {
	s.T.Helper()

	// Ensure we have an initial commit on main if it's the root
	if s.Engine.Trunk() == "main" {
		messages, _ := s.Scene.Repo.ListCurrentBranchCommitMessages()
		if len(messages) == 0 {
			s.WithInitialCommit()
		}
	}

	// We need to create branches in topological order (parents before children).
	// For simplicity in tests, we'll just keep trying until all are created
	// or we stop making progress.
	created := make(map[string]bool)
	created[s.Engine.Trunk()] = true

	for len(created) < len(structure)+1 {
		progress := false
		for branch, parent := range structure {
			if created[branch] {
				continue
			}
			if created[parent] {
				// Create branch
				s.Checkout(parent)
				s.CreateBranch(branch)
				err := s.Scene.Repo.CreateChangeAndCommit("change on "+branch, branch)
				require.NoError(s.T, err)

				// Track it
				err = s.Engine.TrackBranch(context.Background(), branch, parent)
				require.NoError(s.T, err)

				created[branch] = true
				progress = true
			}
		}
		if !progress {
			s.T.Fatalf("could not resolve stack structure: circular dependency or missing parent")
		}
	}

	return s
}

// ExpectStackStructure asserts that the engine's parent-child relationships match the expected map.
func (s *Scenario) ExpectStackStructure(expected map[string]string) *Scenario {
	s.T.Helper()
	for branch, expectedParent := range expected {
		actualParent := s.Engine.GetParent(branch)
		require.Equal(s.T, expectedParent, actualParent, "Parent of %s does not match", branch)
	}
	return s
}

// ExpectBranchFixed asserts that a branch is considered "fixed" (no restack needed) by the engine.
func (s *Scenario) ExpectBranchFixed(branch string) *Scenario {
	s.T.Helper()
	require.True(s.T, s.Engine.IsBranchFixed(context.Background(), branch), "Branch %s should be fixed", branch)
	return s
}

// ExpectBranchNotFixed asserts that a branch is NOT considered "fixed" by the engine.
func (s *Scenario) ExpectBranchNotFixed(branch string) *Scenario {
	s.T.Helper()
	require.False(s.T, s.Engine.IsBranchFixed(context.Background(), branch), "Branch %s should NOT be fixed", branch)
	return s
}

// WithBinaryPath sets the path to the stackit binary for RunCli methods.
func (s *Scenario) WithBinaryPath(path string) *Scenario {
	s.BinaryPath = path
	return s
}

// RunCli executes a stackit CLI command and rebuilds the engine if it exists.
func (s *Scenario) RunCli(args ...string) *Scenario {
	s.T.Helper()
	if s.BinaryPath == "" {
		s.T.Fatal("BinaryPath not set. Call WithBinaryPath first.")
	}
	cmd := exec.Command(s.BinaryPath, args...)
	cmd.Dir = s.Scene.Dir
	cmd.Env = append(os.Environ(), "STACKIT_NON_INTERACTIVE=true")
	output, err := cmd.CombinedOutput()
	require.NoError(s.T, err, "CLI command failed: stackit %v\nOutput: %s", args, string(output))
	if s.Engine != nil {
		return s.Rebuild()
	}
	return s
}

// RunCliAndGetOutput executes a stackit CLI command and returns its output.
func (s *Scenario) RunCliAndGetOutput(args ...string) (string, error) {
	if s.BinaryPath == "" {
		return "", fmt.Errorf("BinaryPath not set")
	}
	cmd := exec.Command(s.BinaryPath, args...)
	cmd.Dir = s.Scene.Dir
	cmd.Env = append(os.Environ(), "STACKIT_NON_INTERACTIVE=true")
	output, err := cmd.CombinedOutput()
	if s.Engine != nil {
		s.Rebuild()
	}
	return string(output), err
}

// RunExpectError executes a stackit CLI command and expects it to fail.
func (s *Scenario) RunExpectError(args ...string) *Scenario {
	s.T.Helper()
	if s.BinaryPath == "" {
		s.T.Fatal("BinaryPath not set")
	}
	cmd := exec.Command(s.BinaryPath, args...)
	cmd.Dir = s.Scene.Dir
	cmd.Env = append(os.Environ(), "STACKIT_NON_INTERACTIVE=true")
	_, err := cmd.CombinedOutput()
	require.Error(s.T, err, "expected CLI command to fail: stackit %v", args)
	if s.Engine != nil {
		return s.Rebuild()
	}
	return s
}

// ExpectBranch asserts that the current branch is as expected.
func (s *Scenario) ExpectBranch(expected string) *Scenario {
	s.T.Helper()
	actual, err := s.Scene.Repo.CurrentBranchName()
	require.NoError(s.T, err)
	require.Equal(s.T, expected, actual)
	return s
}
