package tree

// MockTreeData provides test data for tree rendering.
// It is exported to be used in both tests and the TUI storyboard.
type MockTreeData struct {
	CurrentBranch string
	Trunk         string
	Children      map[string][]string
	Parents       map[string]string
	Fixed         map[string]bool
}

// NewMockTreeData creates a new MockTreeData with sample data.
func NewMockTreeData() *MockTreeData {
	return &MockTreeData{
		CurrentBranch: "feature-2",
		Trunk:         "main",
		Children: map[string][]string{
			"main":      {"feature-1"},
			"feature-1": {"feature-2"},
			"feature-2": {},
		},
		Parents: map[string]string{
			"feature-1": "main",
			"feature-2": "feature-1",
		},
		Fixed: map[string]bool{
			"main":      true,
			"feature-1": true,
			"feature-2": true,
		},
	}
}

// GetChildren returns the children of a branch.
func (m *MockTreeData) GetChildren(branchName string) []string {
	return m.Children[branchName]
}

// GetParent returns the parent of a branch.
func (m *MockTreeData) GetParent(branchName string) string {
	return m.Parents[branchName]
}

// IsTrunk returns whether a branch is the trunk.
func (m *MockTreeData) IsTrunk(branchName string) bool {
	return branchName == m.Trunk
}

// IsBranchFixed returns whether a branch is fixed.
func (m *MockTreeData) IsBranchFixed(branchName string) bool {
	return m.Fixed[branchName]
}
