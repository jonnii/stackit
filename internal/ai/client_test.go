package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const testDiff = "diff --git a/file.go b/file.go\n+new code"

func TestMockClient_GeneratePRDescription(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockResponse("Test Title", "Test Body")

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	title, body, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err != nil {
		t.Fatalf("GeneratePRDescription failed: %v", err)
	}

	if title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", title)
	}

	if body != "Test Body" {
		t.Errorf("Expected body 'Test Body', got '%s'", body)
	}
}

func TestMockClient_ErrorHandling(t *testing.T) {
	mock := NewMockClient()
	expectedErr := errors.New("test error")
	mock.SetMockError(expectedErr)

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	_, _, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestMockClient_NoResponseSet(t *testing.T) {
	mock := NewMockClient()

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	_, _, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err == nil {
		t.Fatal("Expected error when no mock response set, got nil")
	}

	if !strings.Contains(err.Error(), "no mock response set") {
		t.Errorf("Expected error about no mock response, got: %v", err)
	}
}

func TestMockClient_CallCount(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockResponse("Title", "Body")

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	if mock.CallCount() != 0 {
		t.Errorf("Expected call count 0, got %d", mock.CallCount())
	}

	mock.GeneratePRDescription(context.Background(), prCtx)
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1, got %d", mock.CallCount())
	}

	mock.GeneratePRDescription(context.Background(), prCtx)
	if mock.CallCount() != 2 {
		t.Errorf("Expected call count 2, got %d", mock.CallCount())
	}
}

func TestMockClient_LastContext(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockResponse("Title", "Body")

	prCtx1 := &PRContext{
		BranchName: "branch-1",
	}
	prCtx2 := &PRContext{
		BranchName: "branch-2",
	}

	mock.GeneratePRDescription(context.Background(), prCtx1)
	lastCtx := mock.LastContext()
	if lastCtx.BranchName != "branch-1" {
		t.Errorf("Expected last context branch 'branch-1', got '%s'", lastCtx.BranchName)
	}

	mock.GeneratePRDescription(context.Background(), prCtx2)
	lastCtx = mock.LastContext()
	if lastCtx.BranchName != "branch-2" {
		t.Errorf("Expected last context branch 'branch-2', got '%s'", lastCtx.BranchName)
	}
}

func TestMockClient_Reset(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockResponse("Title", "Body")

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	mock.GeneratePRDescription(context.Background(), prCtx)
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1 before reset, got %d", mock.CallCount())
	}

	mock.Reset()

	if mock.CallCount() != 0 {
		t.Errorf("Expected call count 0 after reset, got %d", mock.CallCount())
	}

	if mock.LastContext() != nil {
		t.Error("Expected last context to be nil after reset")
	}

	// Should error after reset
	_, _, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err == nil {
		t.Error("Expected error after reset when no mock response set")
	}
}

func TestMockClient_SetMockResponse_ClearsError(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockError(errors.New("test error"))

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	_, _, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err == nil {
		t.Fatal("Expected error before setting response")
	}

	mock.SetMockResponse("Title", "Body")
	_, _, err = mock.GeneratePRDescription(context.Background(), prCtx)
	if err != nil {
		t.Errorf("Expected no error after setting response, got: %v", err)
	}
}

func TestMockClient_SetMockError_ClearsResponse(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockResponse("Title", "Body")

	prCtx := &PRContext{
		BranchName: "test-branch",
	}

	_, _, err := mock.GeneratePRDescription(context.Background(), prCtx)
	if err != nil {
		t.Fatalf("Expected no error before setting error, got: %v", err)
	}

	mock.SetMockError(errors.New("test error"))
	_, _, err = mock.GeneratePRDescription(context.Background(), prCtx)
	if err == nil {
		t.Error("Expected error after setting error")
	}
}

func TestMockClient_GenerateCommitMessage(t *testing.T) {
	mock := NewMockClient()
	mock.SetMockCommitMessage("feat: add new feature")

	diff := testDiff
	message, err := mock.GenerateCommitMessage(context.Background(), diff)
	if err != nil {
		t.Fatalf("GenerateCommitMessage failed: %v", err)
	}

	if message != "feat: add new feature" {
		t.Errorf("Expected message 'feat: add new feature', got '%s'", message)
	}

	if mock.CommitCallCount() != 1 {
		t.Errorf("Expected commit call count 1, got %d", mock.CommitCallCount())
	}

	if mock.LastDiff() != diff {
		t.Errorf("Expected last diff to match, got '%s'", mock.LastDiff())
	}
}

func TestMockClient_GenerateCommitMessage_Error(t *testing.T) {
	mock := NewMockClient()
	expectedErr := errors.New("commit generation error")
	mock.SetMockCommitError(expectedErr)

	diff := testDiff
	_, err := mock.GenerateCommitMessage(context.Background(), diff)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestMockClient_GenerateCommitMessage_NoResponseSet(t *testing.T) {
	mock := NewMockClient()

	diff := testDiff
	_, err := mock.GenerateCommitMessage(context.Background(), diff)
	if err == nil {
		t.Fatal("Expected error when no mock commit message set, got nil")
	}

	if !strings.Contains(err.Error(), "no mock commit message set") {
		t.Errorf("Expected error about no mock commit message, got: %v", err)
	}
}
