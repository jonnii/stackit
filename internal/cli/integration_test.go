package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

// =============================================================================
// Test Shell - A helper to make tests read like terminal sessions
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

// Log prints a message (useful for documenting test steps)
func (s *TestShell) Log(msg string) *TestShell {
	s.t.Log(msg)
	return s
}

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

// =============================================================================
// Integration Tests - Now read like terminal sessions!
// =============================================================================

func TestIntegrationStackWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("full stack workflow: create, amend, restack, squash", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Build a stack: main -> feature-a -> feature-b -> feature-c
		sh.Log("Creating stacked branches...")
		sh.Write("feature_a", "feature a content").
			Run("create feature-a -m 'Add feature A'").
			OnBranch("feature-a")

		sh.Write("feature_b", "feature b content").
			Run("create feature-b -m 'Add feature B'").
			OnBranch("feature-b")

		sh.Write("feature_c", "feature c content").
			Run("create feature-c -m 'Add feature C'").
			OnBranch("feature-c")

		sh.Run("log --stack").
			OutputContains("feature-a").
			OutputContains("feature-b").
			OutputContains("feature-c")

		// Add commits and amend on feature-a
		sh.Log("Adding commits and amending on feature-a...")
		sh.Checkout("feature-a").
			Commit("feature_a_extra", "additional work").
			CommitCount("main", "feature-a", 2).
			Amend("feature_a_amended", "amended content")

		// Restack to propagate changes
		sh.Log("Restacking upstack branches...")
		sh.Run("restack --upstack")

		// Verify children are still valid
		sh.Checkout("feature-b").
			Run("info").
			OutputContains("feature-b")

		sh.Checkout("feature-c").
			Run("info").
			OutputContains("feature-c")

		// Squash commits on feature-a
		sh.Log("Squashing commits on feature-a...")
		sh.Checkout("feature-a").
			CommitCount("main", "feature-a", 2).
			Run("squash -m 'Feature A complete'").
			CommitCount("main", "feature-a", 1)

		// Verify the squashed commit message
		sh.Run("info").
			OutputContains("Feature A complete")

		// Verify children survived the squash
		sh.Log("Verifying children are still valid after squash...")
		sh.Checkout("feature-b").
			Run("info").
			OutputContains("feature-b")

		sh.Checkout("feature-c").
			Run("info").
			OutputContains("feature-c")

		sh.HasBranches("feature-a", "feature-b", "feature-c", "main")
		sh.Log("✓ Full workflow complete!")
	})

	t.Run("stack workflow with parallel branches", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Create diamond-shaped stack:
		//        main
		//          |
		//      feature-a
		//       /     \
		//   feat-b1  feat-b2
		//               |
		//           feature-c

		sh.Log("Creating diamond-shaped branch structure...")
		sh.Write("a", "feature a").Run("create feature-a -m 'Feature A'")
		sh.Write("b1", "feature b1").Run("create feat-b1 -m 'Feature B1'")

		sh.Checkout("feature-a")
		sh.Write("b2", "feature b2").Run("create feat-b2 -m 'Feature B2'")
		sh.Write("c", "feature c").Run("create feature-c -m 'Feature C'")

		sh.HasBranches("feat-b1", "feat-b2", "feature-a", "feature-c", "main")

		// Amend feature-a and restack everything
		sh.Log("Amending feature-a and restacking...")
		sh.Checkout("feature-a").
			Amend("a_amended", "feature a amended").
			Run("restack --upstack")

		// Verify all branches survived
		sh.Checkout("feat-b1").Run("info").OutputContains("feat-b1")
		sh.Checkout("feat-b2").Run("info").OutputContains("feat-b2")
		sh.Checkout("feature-c").Run("info").OutputContains("feature-c")

		sh.Log("✓ Parallel branch workflow complete!")
	})
}
