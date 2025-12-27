package integration

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

const prStateMerged = "MERGED"

func TestSync(t *testing.T) {
	t.Run("detects and corrects parent from GitHub PR base", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// 1. Create a diamond-like structure:
		// main -> feature-a -> feature-b
		sh.CreateBranch("feature-a").
			CommitChange("file-a", "content-a").
			TrackBranch("feature-a", "main")

		sh.CreateBranch("feature-b").
			CommitChange("file-b", "content-b").
			TrackBranch("feature-b", "feature-a")

		eng := sh.Engine
		mainBranchName := eng.Trunk().GetName()

		// Verify initial parent of feature-b is feature-a
		branchB := eng.GetBranch("feature-b")
		require.Equal(t, "feature-a", eng.GetParent(branchB).GetName())

		// 2. Simulate GitHub PR metadata for feature-b pointing to main instead of feature-a
		t.Log("Simulating changed PR base on GitHub...")
		meta, err := eng.ReadMetadataRef("feature-b")
		require.NoError(t, err)

		if meta.PrInfo == nil {
			meta.PrInfo = &engine.PrInfoPersistence{}
		}
		newBase := mainBranchName
		meta.PrInfo.Base = &newBase

		err = eng.WriteMetadataRef(eng.GetBranch("feature-b"), meta)
		require.NoError(t, err)

		// 3. Run sync (this should update the local parent to match GitHub)
		err = sync.Action(sh.Context, sync.Options{})
		require.NoError(t, err)

		// 4. Verify local parent of feature-b is now main
		branchBAfter := eng.GetBranch("feature-b")
		require.Equal(t, mainBranchName, eng.GetParent(branchBAfter).GetName(), "Local parent should have been updated to match GitHub PR base")
	})

	t.Run("handles consolidation and deletion of merged branches", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// 1. Create a chain of branches: main -> branch-a -> branch-b -> branch-c
		branchNames := []string{"branch-a", "branch-b", "branch-c"}
		currentParent := "main"
		for _, name := range branchNames {
			sh.CreateBranch(name).
				CommitChange(name+"-file", "content").
				TrackBranch(name, currentParent)
			currentParent = name
		}

		eng := sh.Engine
		mainBranchName := eng.Trunk().GetName()

		// Batch read metadata for all individual branches at once
		metas, readErrs := eng.BatchReadMetadataRefs(branchNames)
		for branch, readErr := range readErrs {
			require.NoError(t, readErr, "failed to read metadata for %s", branch)
		}

		// Mark individual branches as merged
		for i, branch := range branchNames {
			meta := metas[branch]
			if meta == nil {
				meta = &engine.Meta{}
			}
			if meta.PrInfo == nil {
				meta.PrInfo = &engine.PrInfoPersistence{}
			}
			prNum := i + 1
			state := prStateMerged
			base := mainBranchName
			meta.PrInfo.Number = &prNum
			meta.PrInfo.State = &state
			meta.PrInfo.Base = &base
			err := eng.WriteMetadataRef(eng.GetBranch(branch), meta)
			require.NoError(t, err)
		}

		// 2. Create a "consolidation" branch representing a Squash & Merge of the whole stack
		// This branch will have the same content as branch-c but only one commit.
		consolidationBranch := "consolidated-feature"
		sh.Checkout("main").
			CreateBranch(consolidationBranch).
			CommitChange("consolidation-file", "consolidated content")

		// Track the consolidation branch (normally consolidation branches aren't tracked,
		// but for the test to work with current clean_branches logic, we need to track it)
		// TODO: Fix clean_branches to handle untracked branches that should be deleted
		parentName := mainBranchName
		err := eng.WriteMetadataRef(eng.GetBranch(consolidationBranch), &engine.Meta{
			ParentBranchName: &parentName,
		})
		require.NoError(t, err)

		// Mark consolidation branch as merged
		meta, err := eng.ReadMetadataRef(consolidationBranch)
		require.NoError(t, err)
		if meta.PrInfo == nil {
			meta.PrInfo = &engine.PrInfoPersistence{}
		}
		prNum := 100
		state := "CLOSED"
		base := mainBranchName
		meta.PrInfo.Number = &prNum
		meta.PrInfo.State = &state
		meta.PrInfo.Base = &base
		err = eng.WriteMetadataRef(eng.GetBranch(consolidationBranch), meta)
		require.NoError(t, err)

		// 3. Run sync (which should call clean_branches)
		err = sync.Action(sh.Context, sync.Options{})
		require.NoError(t, err)

		// 4. Verify all merged branches are deleted
		allLocalBranches, _ := sh.Scene.Repo.GetLocalBranches()
		for _, name := range branchNames {
			require.NotContains(t, allLocalBranches, name, "Merged branch %s should have been deleted", name)
		}
		require.NotContains(t, allLocalBranches, consolidationBranch, "Merged consolidation branch should have been deleted")
	})

	t.Run("handles diamond dependency during sync", func(t *testing.T) {
		sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create: main -> a -> b
		//                \-> c
		sh.CreateBranch("a").CommitChange("a", "a").TrackBranch("a", "main")
		sh.CreateBranch("b").CommitChange("b", "b").TrackBranch("b", "a")
		sh.Checkout("main")
		sh.CreateBranch("c").CommitChange("c", "c").TrackBranch("c", "a")

		// Update GitHub info: 'a' is merged, 'b' is open pointing to 'main', 'c' is open pointing to 'a'
		eng := sh.Engine
		mainBranchName := eng.Trunk().GetName()

		// a: MERGED into main
		metaA, _ := eng.ReadMetadataRef("a")
		if metaA.PrInfo == nil {
			metaA.PrInfo = &engine.PrInfoPersistence{}
		}
		prNumA := 1
		stateA := prStateMerged
		baseA := mainBranchName
		metaA.PrInfo.Number = &prNumA
		metaA.PrInfo.State = &stateA
		metaA.PrInfo.Base = &baseA
		_ = eng.WriteMetadataRef(eng.GetBranch("a"), metaA)

		// b: OPEN pointing to main
		metaB, _ := eng.ReadMetadataRef("b")
		if metaB.PrInfo == nil {
			metaB.PrInfo = &engine.PrInfoPersistence{}
		}
		prNumB := 2
		stateB := "OPEN"
		baseB := mainBranchName
		metaB.PrInfo.Number = &prNumB
		metaB.PrInfo.State = &stateB
		metaB.PrInfo.Base = &baseB
		_ = eng.WriteMetadataRef(eng.GetBranch("b"), metaB)

		// 1. Run sync
		err := sync.Action(sh.Context, sync.Options{})
		require.NoError(t, err)

		// 2. Verify 'a' is deleted
		allLocalBranches, _ := sh.Scene.Repo.GetLocalBranches()
		require.NotContains(t, allLocalBranches, "a")

		// 3. Verify 'b' is reparented to 'main' (because GitHub says so)
		require.Equal(t, "main", eng.GetParent(eng.GetBranch("b")).GetName())

		// 4. Verify 'c' is reparented to 'main' (because 'a' was deleted)
		require.Equal(t, "main", eng.GetParent(eng.GetBranch("c")).GetName())
	})
}

func TestSyncDraftPRs(t *testing.T) {
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create branch-a on main
	sh.CreateBranch("branch-a").
		CommitChange("file-a", "content-a").
		TrackBranch("branch-a", "main")

	eng := sh.Engine
	mainBranchName := eng.Trunk().GetName()

	// 1. Simulate GitHub PR metadata for branch-a being a DRAFT
	t.Log("Simulating DRAFT PR on GitHub...")
	meta, err := eng.ReadMetadataRef("branch-a")
	require.NoError(t, err)

	if meta.PrInfo == nil {
		meta.PrInfo = &engine.PrInfoPersistence{}
	}
	prNum := 1
	state := "OPEN"
	base := mainBranchName
	isDraft := true
	meta.PrInfo.Number = &prNum
	meta.PrInfo.State = &state
	meta.PrInfo.Base = &base
	meta.PrInfo.IsDraft = &isDraft

	err = eng.WriteMetadataRef(eng.GetBranch("branch-a"), meta)
	require.NoError(t, err)

	// 2. Verify it's a draft locally
	branch := eng.GetBranch("branch-a")
	prInfo, _ := eng.GetPrInfo(branch)
	require.True(t, prInfo.IsDraft())

	// 3. Run sync
	err = sync.Action(sh.Context, sync.Options{})
	require.NoError(t, err)

	// 4. Verify branch still exists (drafts shouldn't be deleted)
	allLocalBranches, _ := sh.Scene.Repo.GetLocalBranches()
	require.Contains(t, allLocalBranches, "branch-a")
}

func TestSyncCleanupDiamond(t *testing.T) {
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create: main -> a -> b
	//                \-> c
	sh.CreateBranch("a").CommitChange("a", "a").TrackBranch("a", "main")
	sh.CreateBranch("b").CommitChange("b", "b").TrackBranch("b", "a")
	sh.Checkout("a")
	sh.CreateBranch("c").CommitChange("c", "c").TrackBranch("c", "a")

	eng := sh.Engine
	mainBranchName := eng.Trunk().GetName()

	// Mark 'a' as merged
	metaA, _ := eng.ReadMetadataRef("a")
	if metaA.PrInfo == nil {
		metaA.PrInfo = &engine.PrInfoPersistence{}
	}
	prNum := 1
	state := prStateMerged
	base := mainBranchName
	metaA.PrInfo.Number = &prNum
	metaA.PrInfo.State = &state
	metaA.PrInfo.Base = &base
	_ = eng.WriteMetadataRef(eng.GetBranch("a"), metaA)

	// Mark 'b' as merged
	metaB, _ := eng.ReadMetadataRef("b")
	if metaB.PrInfo == nil {
		metaB.PrInfo = &engine.PrInfoPersistence{}
	}
	prNumB := 2
	stateB := "MERGED"
	baseB := mainBranchName
	metaB.PrInfo.Number = &prNumB
	metaB.PrInfo.State = &stateB
	metaB.PrInfo.Base = &baseB
	_ = eng.WriteMetadataRef(eng.GetBranch("b"), metaB)

	// Run sync
	err := sync.Action(sh.Context, sync.Options{})
	require.NoError(t, err)

	// Verify 'a' and 'b' are deleted, but 'c' remains (not merged)
	allLocalBranches, _ := sh.Scene.Repo.GetLocalBranches()
	require.NotContains(t, allLocalBranches, "a")
	require.NotContains(t, allLocalBranches, "b")
	require.Contains(t, allLocalBranches, "c")

	// Verify 'c' is reparented to 'main'
	require.Equal(t, "main", eng.GetParent(eng.GetBranch("c")).GetName())
}

func TestSyncStaleDraftCleanup(t *testing.T) {
	// Tests the fix for the "stale draft" bug where empty branches
	// weren't being cleaned up if they were ancestors of other branches.
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create: main -> a -> b
	sh.CreateBranch("a").CommitChange("a", "a").TrackBranch("a", "main")
	sh.CreateBranch("b").CommitChange("b", "b").TrackBranch("b", "a")

	// Now 'a' is merged into main (e.g. via squash and merge)
	// and 'b' is NOT merged yet.
	sh.Checkout("main")
	sh.CommitChange("a", "a") // Simulate squash-merge of 'a'

	eng := sh.Engine
	mainBranchName := eng.Trunk().GetName()

	// Mark 'a' as merged in metadata
	metaA, _ := eng.ReadMetadataRef("a")
	if metaA.PrInfo == nil {
		metaA.PrInfo = &engine.PrInfoPersistence{}
	}
	prNum := 1
	state := prStateMerged
	base := mainBranchName
	metaA.PrInfo.Number = &prNum
	metaA.PrInfo.State = &state
	metaA.PrInfo.Base = &base
	_ = eng.WriteMetadataRef(eng.GetBranch("a"), metaA)

	// 'a' is now empty relative to main
	isEmpty, _ := eng.IsBranchEmpty(sh.Context.Context, "a")
	require.True(t, isEmpty)

	// Run sync
	err := sync.Action(sh.Context, sync.Options{})
	require.NoError(t, err)

	// 'a' should be deleted because it's merged and empty
	allLocalBranches, _ := sh.Scene.Repo.GetLocalBranches()
	require.NotContains(t, allLocalBranches, "a")

	// 'b' should be reparented to main
	require.Equal(t, "main", eng.GetParent(eng.GetBranch("b")).GetName())
}
