package actions

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/testhelpers"
)

func TestAnalyzeAction(t *testing.T) {
	scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)

	// Create some staged changes
	filePath := filepath.Join(scene.Dir, "user.go")
	if err := os.WriteFile(filePath, []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := scene.Repo.RunGitCommand("add", "user.go"); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}

	eng, err := engine.NewEngine(scene.Dir)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := &runtime.Context{
		Context:  context.Background(),
		Engine:   eng,
		RepoRoot: scene.Dir,
		Splog:    tui.NewSplog(),
	}

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

	result, err := AnalyzeAction(ctx, opts)
	if err != nil {
		t.Fatalf("AnalyzeAction failed: %v", err)
	}

	if len(result.Layers) != 1 {
		t.Errorf("Expected 1 layer, got %d", len(result.Layers))
	}
	if result.Layers[0].BranchName != "refactor-user" {
		t.Errorf("Expected branch refactor-user, got %s", result.Layers[0].BranchName)
	}
}

func TestCreateStackAction(t *testing.T) {
	scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)

	// Create multiple changes
	if err := os.WriteFile(filepath.Join(scene.Dir, "user.go"), []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scene.Dir, "api.go"), []byte("package api"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := scene.Repo.RunGitCommand("add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}

	eng, err := engine.NewEngine(scene.Dir)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := &runtime.Context{
		Context:  context.Background(),
		Engine:   eng,
		RepoRoot: scene.Dir,
		Splog:    tui.NewSplog(),
	}

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

	err = CreateStackAction(ctx, suggestion, 0)
	if err != nil {
		t.Fatalf("CreateStackAction failed: %v", err)
	}

	// Verify branches were created and tracked
	output, _ := scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "refactor-user")
	if output == "" {
		t.Error("Branch refactor-user should exist")
	}
	output, _ = scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "add-api")
	if output == "" {
		t.Error("Branch add-api should exist")
	}

	parent := eng.GetParent("add-api")
	if parent != "refactor-user" {
		t.Errorf("Expected parent of add-api to be refactor-user, got %s", parent)
	}
}

func TestCreateStackAction_EmptyLayer(t *testing.T) {
	scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)

	// Create some staged changes
	if err := os.WriteFile(filepath.Join(scene.Dir, "user.go"), []byte("package user"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := scene.Repo.RunGitCommand("add", "."); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}

	eng, err := engine.NewEngine(scene.Dir)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := &runtime.Context{
		Context:  context.Background(),
		Engine:   eng,
		RepoRoot: scene.Dir,
		Splog:    tui.NewSplog(),
	}

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

	err = CreateStackAction(ctx, suggestion, 0)
	if err != nil {
		t.Fatalf("CreateStackAction failed: %v", err)
	}

	output, _ := scene.Repo.RunGitCommandAndGetOutput("branch", "--list", "empty-layer")
	if output == "" {
		t.Error("Branch empty-layer should exist")
	}
}
