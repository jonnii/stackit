package git_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestReadMetadataRef(t *testing.T) {
	t.Run("returns nil when metadata does not exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// No metadata exists yet
		meta, err := git.ReadMetadataRef("branch1")
		require.Error(t, err)
		require.Nil(t, meta)
	})

	t.Run("reads existing metadata", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Write metadata
		parentName := "main"
		parentRev := "abc123"
		meta := &git.Meta{
			ParentBranchName:     &parentName,
			ParentBranchRevision: &parentRev,
		}
		err = git.WriteMetadataRef("branch1", meta)
		require.NoError(t, err)

		// Read metadata
		readMeta, err := git.ReadMetadataRef("branch1")
		require.NoError(t, err)
		require.NotNil(t, readMeta)
		require.NotNil(t, readMeta.ParentBranchName)
		require.Equal(t, "main", *readMeta.ParentBranchName)
		require.NotNil(t, readMeta.ParentBranchRevision)
		require.Equal(t, "abc123", *readMeta.ParentBranchRevision)
	})
}

func TestWriteMetadataRef(t *testing.T) {
	t.Run("writes metadata for branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Get actual parent revision
		mainRev, err := scene.Repo.GetRef("main")
		require.NoError(t, err)

		parentName := "main"
		meta := &git.Meta{
			ParentBranchName:     &parentName,
			ParentBranchRevision: &mainRev,
		}

		err = git.WriteMetadataRef("branch1", meta)
		require.NoError(t, err)

		// Verify metadata was written
		readMeta, err := git.ReadMetadataRef("branch1")
		require.NoError(t, err)
		require.NotNil(t, readMeta)
		require.Equal(t, "main", *readMeta.ParentBranchName)
	})
}

func TestDeleteMetadataRef(t *testing.T) {
	t.Run("deletes metadata for branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Write metadata
		parentName := "main"
		meta := &git.Meta{
			ParentBranchName: &parentName,
		}
		err = git.WriteMetadataRef("branch1", meta)
		require.NoError(t, err)

		// Delete metadata
		err = git.DeleteMetadataRef("branch1")
		require.NoError(t, err)

		// Verify metadata is gone
		readMeta, err := git.ReadMetadataRef("branch1")
		require.Error(t, err)
		require.Nil(t, readMeta)
	})
}

func TestGetMetadataRefList(t *testing.T) {
	t.Run("returns list of branches with metadata", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branches
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Write metadata for both branches
		parentName := "main"
		meta1 := &git.Meta{ParentBranchName: &parentName}
		err = git.WriteMetadataRef("branch1", meta1)
		require.NoError(t, err)

		meta2 := &git.Meta{ParentBranchName: &parentName}
		err = git.WriteMetadataRef("branch2", meta2)
		require.NoError(t, err)

		// Get metadata list
		refList, err := git.GetMetadataRefList()
		require.NoError(t, err)
		require.Contains(t, refList, "branch1")
		require.Contains(t, refList, "branch2")
	})
}
