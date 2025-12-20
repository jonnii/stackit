package ai

import (
	"context"
	"fmt"
	"sync"
)

// MockClient is a mock implementation of AIClient for testing purposes.
// It allows setting predefined responses and errors without making actual API calls.
type MockClient struct {
	mu                sync.Mutex
	mockTitle         string
	mockBody          string
	mockCommitMessage string
	mockError         error
	mockCommitError   error
	callCount         int
	commitCallCount   int
	lastContext       *PRContext
	callContexts      []*PRContext
	lastDiff          string
}

// NewMockClient creates a new MockClient instance.
func NewMockClient() *MockClient {
	return &MockClient{
		callContexts: make([]*PRContext, 0),
	}
}

// GeneratePRDescription implements AIClient interface.
// Returns the mock response if set, otherwise returns an error.
func (m *MockClient) GeneratePRDescription(ctx context.Context, prContext *PRContext) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.lastContext = prContext
	// Create a copy of context for history
	contextCopy := *prContext
	m.callContexts = append(m.callContexts, &contextCopy)

	if m.mockError != nil {
		return "", "", m.mockError
	}

	if m.mockTitle == "" && m.mockBody == "" {
		return "", "", fmt.Errorf("no mock response set, use SetMockResponse()")
	}

	return m.mockTitle, m.mockBody, nil
}

// SetMockResponse sets the mock response to return for GeneratePRDescription.
func (m *MockClient) SetMockResponse(title, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockTitle = title
	m.mockBody = body
	m.mockError = nil
}

// SetMockError sets the mock error to return for GeneratePRDescription.
func (m *MockClient) SetMockError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockError = err
	m.mockTitle = ""
	m.mockBody = ""
}

// Reset clears all mock state.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockTitle = ""
	m.mockBody = ""
	m.mockCommitMessage = ""
	m.mockError = nil
	m.mockCommitError = nil
	m.callCount = 0
	m.commitCallCount = 0
	m.lastContext = nil
	m.lastDiff = ""
	m.callContexts = make([]*PRContext, 0)
}

// GenerateCommitMessage implements AIClient interface.
// Returns the mock commit message if set, otherwise returns an error.
func (m *MockClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commitCallCount++
	m.lastDiff = diff

	if m.mockCommitError != nil {
		return "", m.mockCommitError
	}

	if m.mockCommitMessage == "" {
		return "", fmt.Errorf("no mock commit message set, use SetMockCommitMessage()")
	}

	return m.mockCommitMessage, nil
}

// SetMockCommitMessage sets the mock commit message to return for GenerateCommitMessage.
func (m *MockClient) SetMockCommitMessage(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockCommitMessage = message
	m.mockCommitError = nil
}

// SetMockCommitError sets the mock error to return for GenerateCommitMessage.
func (m *MockClient) SetMockCommitError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockCommitError = err
	m.mockCommitMessage = ""
}

// CommitCallCount returns the number of times GenerateCommitMessage has been called.
func (m *MockClient) CommitCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitCallCount
}

// LastDiff returns the last diff passed to GenerateCommitMessage.
func (m *MockClient) LastDiff() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastDiff
}

// CallCount returns the number of times GeneratePRDescription has been called.
func (m *MockClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// LastContext returns the last PRContext passed to GeneratePRDescription.
func (m *MockClient) LastContext() *PRContext {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastContext
}

// CallContexts returns all PRContexts passed to GeneratePRDescription.
func (m *MockClient) CallContexts() []*PRContext {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callContexts
}
