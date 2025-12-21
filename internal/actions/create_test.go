package actions_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCreateAction_AI(t *testing.T) {
	t.Run("generates commit message and branch name with --ai", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a tracked file first, then modify it unstaged
		err := s.Scene.Repo.CreateChangeAndCommit("initial content", "test")
		require.NoError(t, err)

		// Now create unstaged changes to the tracked file
		err = s.Scene.Repo.CreateChange("modified content", "test", true)
		require.NoError(t, err)

		// Create mock AI client
		mockClient := ai.NewMockClient()
		mockClient.SetMockCommitMessage("feat: add new feature")

		opts := actions.CreateOptions{
			AI:       true,
			AIClient: mockClient,
		}

		err = actions.CreateAction(s.Context, opts)
		require.NoError(t, err)

		// Verify AI was called
		require.Equal(t, 1, mockClient.CommitCallCount(), "AI should be called once")

		// Verify branch was created with name generated from AI message
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Contains(t, currentBranch, "add-new-feature", "Branch name should be generated from AI commit message")

		// Verify commit was created with AI message
		commits, err := s.Scene.Repo.ListCurrentBranchCommitMessages()
		require.NoError(t, err)
		require.Contains(t, commits, "feat: add new feature")
	})

	t.Run("fails when AI generation fails", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a tracked file first, then modify it unstaged
		err := s.Scene.Repo.CreateChangeAndCommit("initial content", "test")
		require.NoError(t, err)

		// Now create unstaged changes to the tracked file
		err = s.Scene.Repo.CreateChange("modified content", "test", true)
		require.NoError(t, err)

		// Create mock AI client with error
		mockClient := ai.NewMockClient()
		mockClient.SetMockCommitError(fmt.Errorf("AI service unavailable"))

		opts := actions.CreateOptions{
			AI:       true,
			AIClient: mockClient,
		}

		err = actions.CreateAction(s.Context, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to generate commit message with AI")
	})

	t.Run("fails when no changes available for AI", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create mock AI client
		mockClient := ai.NewMockClient()
		mockClient.SetMockCommitMessage("feat: add feature")

		opts := actions.CreateOptions{
			AI:       true,
			AIClient: mockClient,
		}

		// No changes staged or unstaged
		err := actions.CreateAction(s.Context, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no changes to commit")
	})

	t.Run("works with branch name and --ai (no message generation)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create mock AI client (should not be called when branch name provided)
		mockClient := ai.NewMockClient()

		opts := actions.CreateOptions{
			BranchName: "feature",
			AI:         true,
			AIClient:   mockClient,
		}

		err := actions.CreateAction(s.Context, opts)
		require.NoError(t, err)

		// Verify AI was not called (branch name provided, so no message generation)
		require.Equal(t, 0, mockClient.CommitCallCount(), "AI should not be called when branch name provided")

		// Verify branch was created
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("generates branch name from AI commit message when no branch name provided", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Set up branch name pattern config
		err := s.Scene.Repo.RunGitCommand("config", "--local", "stackit.branchNamePattern", "{message}")
		require.NoError(t, err)

		// Create a tracked file first, then modify it unstaged
		err = s.Scene.Repo.CreateChangeAndCommit("initial content", "test")
		require.NoError(t, err)

		// Now create unstaged changes to the tracked file
		err = s.Scene.Repo.CreateChange("modified content", "test", true)
		require.NoError(t, err)

		// Create mock AI client with a realistic commit message
		mockClient := ai.NewMockClient()
		mockClient.SetMockCommitMessage("feat: add new authentication feature")

		opts := actions.CreateOptions{
			AI:       true,
			AIClient: mockClient,
		}

		err = actions.CreateAction(s.Context, opts)
		require.NoError(t, err, "CreateAction should succeed with AI-generated commit message")

		// Verify AI was called
		require.Equal(t, 1, mockClient.CommitCallCount(), "AI should be called once")

		// Verify branch was created with name generated from AI message
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Contains(t, currentBranch, "add-new-authentication-feature", "Branch name should be generated from AI commit message")

		// Verify commit was created with AI message
		commits, err := s.Scene.Repo.ListCurrentBranchCommitMessages()
		require.NoError(t, err)
		require.Contains(t, commits, "feat: add new authentication feature")
	})

	// Note: Markdown stripping is tested in the ai package (TestStripMarkdownCodeBlocks).
	// The mock client returns messages directly without markdown, so we don't test
	// markdown stripping here. The real CursorAgentClient handles markdown stripping
	// in callCursorAgentCLI and callCursorAPI methods.
}
