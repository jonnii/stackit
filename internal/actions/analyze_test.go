package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestAnalyzeAction(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create some staged changes
	filePath := filepath.Join(s.Scene.Dir, "user.go")
	if err := os.WriteFile(filePath, []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	s.RunGit("add", "user.go")

	mockAI := ai.NewMockClient()
	suggestion := &ai.StackSuggestion{
		Layers: []ai.StackLayer{
			{
				BranchName:    "refactor-user",
				Files:         []string{"user.go"},
				Rationale:     "Refactoring user",
				CommitMessage: "refactor: update user",
			},
		},
	}
	mockAI.SetMockSuggestion(suggestion)

	opts := AnalyzeOptions{
		AIClient: mockAI,
	}

	result, err := AnalyzeAction(s.Context, opts)
	require.NoError(t, err)

	if len(result.Layers) != 1 {
		t.Errorf("Expected 1 layer, got %d", len(result.Layers))
	}
	if result.Layers[0].BranchName != "refactor-user" {
		t.Errorf("Expected branch refactor-user, got %s", result.Layers[0].BranchName)
	}
}

func TestCreateStackAction(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create multiple changes
	if err := os.WriteFile(filepath.Join(s.Scene.Dir, "user.go"), []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.Scene.Dir, "api.go"), []byte("package api"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	s.RunGit("add", ".")

	suggestion := &ai.StackSuggestion{
		Layers: []ai.StackLayer{
			{
				BranchName:    "refactor-user",
				Files:         []string{"user.go"},
				Rationale:     "Refactoring user",
				CommitMessage: "refactor: update user",
			},
			{
				BranchName:    "add-api",
				Files:         []string{"api.go"},
				Rationale:     "Adding API",
				CommitMessage: "feat: add api",
			},
		},
	}

	err := CreateStackAction(s.Context, suggestion, 0)
	require.NoError(t, err)

	// Verify branches were created and tracked
	output, _ := s.Scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "refactor-user")
	if output == "" {
		t.Error("Branch refactor-user should exist")
	}
	output, _ = s.Scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "add-api")
	if output == "" {
		t.Error("Branch add-api should exist")
	}

	parent := s.Engine.GetParent("add-api")
	if parent != "refactor-user" {
		t.Errorf("Expected parent of add-api to be refactor-user, got %s", parent)
	}
}

func TestCreateStackAction_EmptyLayer(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create some staged changes
	if err := os.WriteFile(filepath.Join(s.Scene.Dir, "user.go"), []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	s.RunGit("add", ".")

	suggestion := &ai.StackSuggestion{
		Layers: []ai.StackLayer{
			{
				BranchName:    "empty-layer",
				Files:         []string{}, // Empty files
				Rationale:     "Empty",
				CommitMessage: "", // Empty message
			},
		},
	}

	err := CreateStackAction(s.Context, suggestion, 0)
	require.NoError(t, err)

	output, _ := s.Scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "empty-layer")
	if output == "" {
		t.Error("Branch empty-layer should exist")
	}
}
