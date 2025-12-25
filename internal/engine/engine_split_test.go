package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplySplitToCommits(t *testing.T) {
	t.Run("creates branches at specified commit points", func(t *testing.T) {
		// This test needs a real git repository and engine
		// For now, create a basic structure test
		t.Skip("Requires full engine setup - implement when engine test infrastructure is available")
	})

	t.Run("validates branch names and points match", func(t *testing.T) {
		// Create a mock engine for this validation test
		eng := &engineImpl{}

		opts := ApplySplitOptions{
			BranchToSplit: "feature",
			BranchNames:   []string{"branch1"}, // 1 name
			BranchPoints:  []int{0, 1},         // 2 points
		}

		err := eng.ApplySplitToCommits(context.Background(), opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid number of branch names")
	})

	t.Run("fails when branch has no parent", func(t *testing.T) {
		t.Skip("Requires engine with metadata setup")
	})

	t.Run("successfully applies split with valid inputs", func(t *testing.T) {
		t.Skip("Requires full git repository and engine setup")
	})
}

// Test helper functions

func TestContains(t *testing.T) {
	// Test the contains helper function used in ApplySplitToCommits
	require.True(t, contains([]string{"a", "b", "c"}, "b"))
	require.False(t, contains([]string{"a", "b", "c"}, "d"))
	require.False(t, contains([]string{}, "a"))
}
