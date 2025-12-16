package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

// =============================================================================
// Test Shell - A helper to make integration tests read like terminal sessions
// =============================================================================

// TestShell wraps a test scene and provides a fluent interface for running
// commands. Tests using this read like a series of terminal commands.
type TestShell struct {
	t          *testing.T
	scene      *testhelpers.Scene
	binaryPath string
	lastOutput string
}

// NewTestShell creates a shell-like test environment with an initialized repo.
func NewTestShell(t *testing.T, binaryPath string) *TestShell {
	t.Helper()
	scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
		return s.Repo.CreateChangeAndCommit("initial", "init")
	})
	return &TestShell{t: t, scene: scene, binaryPath: binaryPath}
}

// NewTestShellWithRemote creates a shell-like test environment with a local bare repo as "origin".
// This is useful for testing sync workflows that require a remote.
func NewTestShellWithRemote(t *testing.T, binaryPath string) *TestShell {
	t.Helper()

	// Create a bare repository to act as the remote
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", remoteDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to create bare repo: %s", string(output))

	// Create the scene with the remote set up
	scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
		// Create initial commit
		if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
			return err
		}
		// Add the bare repo as origin
		cmd := exec.Command("git", "remote", "add", "origin", remoteDir)
		cmd.Dir = s.Dir
		if err := cmd.Run(); err != nil {
			return err
		}
		// Push main to origin
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = s.Dir
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	})
	return &TestShell{t: t, scene: scene, binaryPath: binaryPath}
}

// Scene returns the underlying test scene for direct access when needed.
func (s *TestShell) Scene() *testhelpers.Scene {
	return s.scene
}

// Dir returns the working directory of the test shell.
func (s *TestShell) Dir() string {
	return s.scene.Dir
}

// =============================================================================
// Command Execution
// =============================================================================

// Run executes a stackit CLI command (e.g., "create feature-a -m 'Add feature'")
func (s *TestShell) Run(args string) *TestShell {
	s.t.Helper()
	parts := splitArgs(args)
	cmd := exec.Command(s.binaryPath, parts...)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	s.lastOutput = string(output)
	require.NoError(s.t, err, "$ stackit %s\n%s", args, s.lastOutput)
	return s
}

// RunExpectError executes a stackit CLI command and expects it to fail.
func (s *TestShell) RunExpectError(args string) *TestShell {
	s.t.Helper()
	parts := splitArgs(args)
	cmd := exec.Command(s.binaryPath, parts...)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	s.lastOutput = string(output)
	require.Error(s.t, err, "$ stackit %s (expected error)\n%s", args, s.lastOutput)
	return s
}

// Git executes a raw git command (use sparingly - prefer stackit commands)
func (s *TestShell) Git(args string) *TestShell {
	s.t.Helper()
	parts := splitArgs(args)
	cmd := exec.Command("git", parts...)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	s.lastOutput = string(output)
	require.NoError(s.t, err, "$ git %s\n%s", args, s.lastOutput)
	return s
}

// =============================================================================
// Navigation Shortcuts
// =============================================================================

// Checkout switches to a branch using stackit checkout
func (s *TestShell) Checkout(branch string) *TestShell {
	s.t.Helper()
	return s.Run("checkout " + branch)
}

// Top navigates to the top of the current stack
func (s *TestShell) Top() *TestShell {
	s.t.Helper()
	return s.Run("top")
}

// Bottom navigates to the bottom of the current stack
func (s *TestShell) Bottom() *TestShell {
	s.t.Helper()
	return s.Run("bottom")
}

// =============================================================================
// File Operations
// =============================================================================

// Write creates/modifies a file and stages it (simulates editing a file)
func (s *TestShell) Write(filename, content string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChange(content, filename, false)
	require.NoError(s.t, err, "failed to write %s", filename)
	return s
}

// Amend modifies a file and amends the last commit
func (s *TestShell) Amend(filename, content string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChangeAndAmend(content, filename)
	require.NoError(s.t, err, "failed to amend with %s", filename)
	return s
}

// Commit creates a file change and commits it
func (s *TestShell) Commit(filename, message string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChangeAndCommit(message, filename)
	require.NoError(s.t, err, "failed to commit %s", filename)
	return s
}

// =============================================================================
// Output Inspection
// =============================================================================

// Output returns the last command's output
func (s *TestShell) Output() string {
	return s.lastOutput
}

// OutputContains asserts the last output contains the given string
func (s *TestShell) OutputContains(substr string) *TestShell {
	s.t.Helper()
	require.Contains(s.t, s.lastOutput, substr)
	return s
}

// OutputNotContains asserts the last output does NOT contain the given string
func (s *TestShell) OutputNotContains(substr string) *TestShell {
	s.t.Helper()
	require.NotContains(s.t, s.lastOutput, substr)
	return s
}

// =============================================================================
// Assertions
// =============================================================================

// OnBranch asserts we're on the expected branch
func (s *TestShell) OnBranch(expected string) *TestShell {
	s.t.Helper()
	branch, err := s.scene.Repo.CurrentBranchName()
	require.NoError(s.t, err)
	require.Equal(s.t, expected, branch)
	return s
}

// HasBranches asserts the repo has exactly these branches
func (s *TestShell) HasBranches(branches ...string) *TestShell {
	s.t.Helper()
	testhelpers.ExpectBranches(s.t, s.scene.Repo, branches)
	return s
}

// CommitCount asserts the number of commits between two refs
func (s *TestShell) CommitCount(from, to string, expected int) *TestShell {
	s.t.Helper()
	cmd := exec.Command("git", "log", "--oneline", from+".."+to)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	require.NoError(s.t, err)
	actual := countNonEmptyLines(string(output))
	require.Equal(s.t, expected, actual, "expected %d commits between %s..%s, got %d", expected, from, to, actual)
	return s
}

// =============================================================================
// Logging
// =============================================================================

// Log prints a message (useful for documenting test steps)
func (s *TestShell) Log(msg string) *TestShell {
	s.t.Log(msg)
	return s
}

// =============================================================================
// Utility Functions
// =============================================================================

// splitArgs splits a command string into args, respecting quotes
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// countNonEmptyLines counts lines that have non-whitespace content
func countNonEmptyLines(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
