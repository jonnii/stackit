package create

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCreateAction_Stdin(t *testing.T) {
	t.Run("reads commit message from stdin in non-interactive mode", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Create a change to stage
		err := s.Scene.Repo.CreateChange("staged content", "test-file", false)
		require.NoError(t, err)

		// Mock stdin
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		r, w, _ := os.Pipe()
		os.Stdin = r

		expectedMessage := "feat: commit message from stdin"
		go func() {
			_, _ = w.Write([]byte(expectedMessage))
			_ = w.Close()
		}()

		// Scenario already sets STACKIT_NON_INTERACTIVE=true

		opts := Options{}
		err = Action(s.Context, opts)
		require.NoError(t, err)

		// Verify branch was created with name generated from stdin message
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Contains(t, currentBranch, "commit-message-from-stdin")

		// Verify commit message
		commits, err := s.Scene.Repo.ListCurrentBranchCommitMessages()
		require.NoError(t, err)
		require.Contains(t, commits, expectedMessage)
	})
}

func TestCreateAction_Insert(t *testing.T) {
	t.Run("inserts branch between parent and children", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// 1. Create child1 on main
		err := s.Scene.Repo.CreateChange("child1 content", "file1", false)
		require.NoError(t, err)
		opts1 := Options{
			BranchName: "child1",
			Message:    "Add child1",
		}
		err = Action(s.Context, opts1)
		require.NoError(t, err)

		// 2. Go back to main
		err = s.Scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// 3. Create 'inserted' branch with --insert
		err = s.Scene.Repo.CreateChange("inserted content", "file2", false)
		require.NoError(t, err)
		opts2 := Options{
			BranchName: "inserted",
			Message:    "Add inserted",
			Insert:     true,
		}
		err = Action(s.Context, opts2)
		require.NoError(t, err)

		// 4. Verify metadata relationships
		eng := s.Context.Engine
		branchparentInserted := eng.GetBranch("inserted")
		parentInserted := eng.GetParent(branchparentInserted)
		require.NotNil(t, parentInserted)
		require.Equal(t, "main", parentInserted.Name)
		branchparentChild1 := eng.GetBranch("child1")
		parentChild1 := eng.GetParent(branchparentChild1)
		require.NotNil(t, parentChild1)
		require.Equal(t, "inserted", parentChild1.Name)

		// 5. Verify physical relationship (child1 should have been restacked onto inserted)
		isAncestor, err := s.Scene.Repo.IsAncestor("inserted", "child1")
		require.NoError(t, err)
		require.True(t, isAncestor, "inserted should be an ancestor of child1 in git history")
	})

	t.Run("inserts branch in the middle of a stack", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// 1. Create stack: main -> child1 -> child2
		err := s.Scene.Repo.CreateChange("child1 content", "file1", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child1", Message: "Add child1"})
		require.NoError(t, err)

		err = s.Scene.Repo.CreateChange("child2 content", "file2", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child2", Message: "Add child2"})
		require.NoError(t, err)

		// 2. Go to child1
		err = s.Scene.Repo.CheckoutBranch("child1")
		require.NoError(t, err)

		// Rebuild engine to ensure it knows we're on child1
		err = s.Context.Engine.Rebuild(s.Context.Engine.Trunk().Name)
		require.NoError(t, err)

		// 3. Insert 'inserted' after child1
		err = s.Scene.Repo.CreateChange("inserted content", "file3", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{
			BranchName: "inserted",
			Message:    "Add inserted",
			Insert:     true,
		})
		require.NoError(t, err)

		// 4. Verify relationships
		eng := s.Context.Engine
		branchparentInserted := eng.GetBranch("inserted")
		parentInserted := eng.GetParent(branchparentInserted)
		require.NotNil(t, parentInserted)
		require.Equal(t, "child1", parentInserted.Name)
		branchparentChild2 := eng.GetBranch("child2")
		parentChild2 := eng.GetParent(branchparentChild2)
		require.NotNil(t, parentChild2)
		require.Equal(t, "inserted", parentChild2.Name)

		// 5. Verify physical relationship
		isAncestor, err := s.Scene.Repo.IsAncestor("inserted", "child2")
		require.NoError(t, err)
		require.True(t, isAncestor, "inserted should be an ancestor of child2")
	})

	t.Run("inserts branch into a branching stack (multiple children)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// 1. Create two children from main: main -> child1, main -> child2
		err := s.Scene.Repo.CreateChange("child1 content", "file1", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child1", Message: "Add child1"})
		require.NoError(t, err)

		err = s.Scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		err = s.Scene.Repo.CreateChange("child2 content", "file2", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child2", Message: "Add child2"})
		require.NoError(t, err)

		// 2. Go back to main
		err = s.Scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// 3. Insert 'inserted' after main
		err = s.Scene.Repo.CreateChange("inserted content", "file3", false)
		require.NoError(t, err)
		// Non-interactive mode should move all children by default
		err = Action(s.Context, Options{
			BranchName: "inserted",
			Message:    "Add inserted",
			Insert:     true,
		})
		require.NoError(t, err)

		// 4. Verify relationships
		eng := s.Context.Engine
		branchparentInserted := eng.GetBranch("inserted")
		parentInserted := eng.GetParent(branchparentInserted)
		require.NotNil(t, parentInserted)
		require.Equal(t, "main", parentInserted.Name)
		branchparentChild1 := eng.GetBranch("child1")
		parentChild1 := eng.GetParent(branchparentChild1)
		require.NotNil(t, parentChild1)
		require.Equal(t, "inserted", parentChild1.Name)
		branchparentChild2 := eng.GetBranch("child2")
		parentChild2 := eng.GetParent(branchparentChild2)
		require.NotNil(t, parentChild2)
		require.Equal(t, "inserted", parentChild2.Name)

		// 5. Verify physical relationships
		isAncestor, err := s.Scene.Repo.IsAncestor("inserted", "child1")
		require.NoError(t, err)
		require.True(t, isAncestor, "inserted should be an ancestor of child1")

		isAncestor, err = s.Scene.Repo.IsAncestor("inserted", "child2")
		require.NoError(t, err)
		require.True(t, isAncestor, "inserted should be an ancestor of child2")
	})

	t.Run("inserts branch into a branching stack selecting only one child", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// 1. Create two children from main: main -> child1, main -> child2
		err := s.Scene.Repo.CreateChange("child1 content", "file1", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child1", Message: "Add child1"})
		require.NoError(t, err)

		err = s.Scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		err = s.Scene.Repo.CreateChange("child2 content", "file2", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{BranchName: "child2", Message: "Add child2"})
		require.NoError(t, err)

		// 2. Go back to main
		err = s.Scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// 3. Insert 'inserted' after main, but only move 'child1'
		err = s.Scene.Repo.CreateChange("inserted content", "file3", false)
		require.NoError(t, err)
		err = Action(s.Context, Options{
			BranchName:       "inserted",
			Message:          "Add inserted",
			Insert:           true,
			SelectedChildren: []string{"child1"},
		})
		require.NoError(t, err)

		// 4. Verify relationships
		eng := s.Context.Engine
		branchparentInserted := eng.GetBranch("inserted")
		parentInserted := eng.GetParent(branchparentInserted)
		require.NotNil(t, parentInserted)
		require.Equal(t, "main", parentInserted.Name)
		branchparentChild1 := eng.GetBranch("child1")
		parentChild1 := eng.GetParent(branchparentChild1)
		require.NotNil(t, parentChild1)
		require.Equal(t, "inserted", parentChild1.Name, "child1 should have been moved to inserted")
		branchparentChild2 := eng.GetBranch("child2")
		parentChild2 := eng.GetParent(branchparentChild2)
		require.NotNil(t, parentChild2)
		require.Equal(t, "main", parentChild2.Name, "child2 should have remained a child of main")

		// 5. Verify physical relationships
		isAncestor, err := s.Scene.Repo.IsAncestor("inserted", "child1")
		require.NoError(t, err)
		require.True(t, isAncestor, "inserted should be an ancestor of child1")

		isAncestor, err = s.Scene.Repo.IsAncestor("inserted", "child2")
		require.NoError(t, err)
		require.False(t, isAncestor, "inserted should NOT be an ancestor of child2")
	})
}
