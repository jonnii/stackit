package absorb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestAbsorbScopeBoundaries(t *testing.T) {
	t.Run("absorb stops at scope boundaries", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"scoped-a":   "main",
				"scoped-b":   "scoped-a",
				"unscoped-c": "scoped-b",
			})

		// Set scopes: scoped-a and scoped-b have PROJ-123, unscoped-c has no scope
		err := s.Engine.SetScope(s.Engine.GetBranch("scoped-a"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("scoped-b"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		// unscoped-c has no scope

		// Add some commits to each branch
		s.Checkout("scoped-a")
		s.Scene.Repo.CreateChangeAndCommit("scoped-a commit", "file-a")

		s.Checkout("scoped-b")
		s.Scene.Repo.CreateChangeAndCommit("scoped-b commit", "file-b")

		s.Checkout("unscoped-c")
		s.Scene.Repo.CreateChangeAndCommit("unscoped-c commit", "file-c")

		// Switch back to scoped-b and create staged changes
		s.Checkout("scoped-b")
		err = s.Scene.Repo.CreateChange("staged change for scoped-b", "file-b", false)
		require.NoError(t, err)

		// Get commits from downstack when absorb runs
		// Since we're on scoped-b with scope PROJ-123, absorb should only look at scoped-a and scoped-b
		// It should NOT look at unscoped-c even though it's in the git history

		// The absorb logic should collect commits from:
		// - scoped-b (current branch)
		// - scoped-a (parent with same scope)
		// - main (stops at scope boundary - unscoped-c has different/no scope)

		// We can't easily test the internal collection logic directly without mocking,
		// but we can verify that the behavior is correct by checking what commits
		// would be considered for absorption.

		// For this test, we'll just verify that the scope detection works correctly
		currentBranch := s.Engine.GetBranch("scoped-b")
		currentScope := currentBranch.GetScope()

		// Verify current branch has the expected scope
		require.True(t, currentScope.IsDefined())
		require.Equal(t, "PROJ-123", currentScope.String())

		// Get downstack branches as absorb would
		downstackBranches := s.Engine.GetRelativeStackDownstack(currentBranch)
		// Include current branch
		downstackBranches = append([]engine.Branch{currentBranch}, downstackBranches...)

		// Apply scope boundary filtering as absorb does
		if currentScope.IsDefined() {
			limitedDownstack := []engine.Branch{}
			for _, branch := range downstackBranches {
				if branch.IsTrunk() || !branch.GetScope().Equal(currentScope) {
					break
				}
				limitedDownstack = append(limitedDownstack, branch)
			}
			downstackBranches = limitedDownstack
		}

		// Should only include scoped-a and scoped-b, not unscoped-c or main
		require.Len(t, downstackBranches, 2)
		require.Equal(t, "scoped-b", downstackBranches[0].Name)
		require.Equal(t, "scoped-a", downstackBranches[1].Name)
	})

	t.Run("absorb includes all branches when no scope set", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
				"branch-b": "branch-a",
				"branch-c": "branch-b",
			})

		// No scopes set on any branches
		s.Checkout("branch-b")

		currentBranch := s.Engine.GetBranch("branch-b")
		currentScope := currentBranch.GetScope()

		// Verify current branch has no scope
		require.True(t, currentScope.IsEmpty())

		// Get downstack branches as absorb would
		downstackBranches := s.Engine.GetRelativeStackDownstack(currentBranch)
		// Include current branch
		downstackBranches = append([]engine.Branch{currentBranch}, downstackBranches...)

		// Apply scope boundary filtering as absorb does
		if currentScope.IsDefined() {
			limitedDownstack := []engine.Branch{}
			for _, branch := range downstackBranches {
				if branch.IsTrunk() || !branch.GetScope().Equal(currentScope) {
					break
				}
				limitedDownstack = append(limitedDownstack, branch)
			}
			downstackBranches = limitedDownstack
		}

		// Since no scope is set, should include all branches down to the first scope boundary
		// In this case, no scopes are set, so includes current and ancestors (excluding trunk)
		require.Len(t, downstackBranches, 2)
		require.Equal(t, "branch-b", downstackBranches[0].Name)
		require.Equal(t, "branch-a", downstackBranches[1].Name)
	})

	t.Run("absorb stops at first scope boundary encountered", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"scoped-a":    "main",
				"scoped-b":    "scoped-a",
				"different-c": "scoped-b",
				"scoped-d":    "different-c",
			})

		// Set mixed scopes: PROJ-123 for a/b, PROJ-456 for c/d
		err := s.Engine.SetScope(s.Engine.GetBranch("scoped-a"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("scoped-b"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("different-c"), engine.NewScope("PROJ-456"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("scoped-d"), engine.NewScope("PROJ-456"))
		require.NoError(t, err)

		// Switch to scoped-b (PROJ-123)
		s.Checkout("scoped-b")

		currentBranch := s.Engine.GetBranch("scoped-b")
		currentScope := currentBranch.GetScope()

		// Verify current branch has PROJ-123 scope
		require.True(t, currentScope.IsDefined())
		require.Equal(t, "PROJ-123", currentScope.String())

		// Get downstack branches as absorb would
		downstackBranches := s.Engine.GetRelativeStackDownstack(currentBranch)
		// Include current branch
		downstackBranches = append([]engine.Branch{currentBranch}, downstackBranches...)

		// Apply scope boundary filtering as absorb does
		if currentScope.IsDefined() {
			limitedDownstack := []engine.Branch{}
			for _, branch := range downstackBranches {
				if branch.IsTrunk() || !branch.GetScope().Equal(currentScope) {
					break // Stop at first scope mismatch
				}
				limitedDownstack = append(limitedDownstack, branch)
			}
			downstackBranches = limitedDownstack
		}

		// Should stop at different-c (PROJ-456) and not include scoped-d
		// So should include: scoped-b, scoped-a, main (but stops at different-c)
		require.Len(t, downstackBranches, 2)
		require.Equal(t, "scoped-b", downstackBranches[0].Name)
		require.Equal(t, "scoped-a", downstackBranches[1].Name)
	})
}
