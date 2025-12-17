package output

import (
	"strings"
	"testing"
)

// mockTreeData provides test data for tree rendering
type mockTreeData struct {
	currentBranch string
	trunk         string
	children      map[string][]string
	parents       map[string]string
	fixed         map[string]bool
}

func newMockTreeData() *mockTreeData {
	return &mockTreeData{
		currentBranch: "feature-2",
		trunk:         "main",
		children: map[string][]string{
			"main":      {"feature-1"},
			"feature-1": {"feature-2"},
			"feature-2": {},
		},
		parents: map[string]string{
			"feature-1": "main",
			"feature-2": "feature-1",
		},
		fixed: map[string]bool{
			"main":      true,
			"feature-1": true,
			"feature-2": true,
		},
	}
}

func (m *mockTreeData) getChildren(branchName string) []string {
	return m.children[branchName]
}

func (m *mockTreeData) getParent(branchName string) string {
	return m.parents[branchName]
}

func (m *mockTreeData) isTrunk(branchName string) bool {
	return branchName == m.trunk
}

func (m *mockTreeData) isBranchFixed(branchName string) bool {
	return m.fixed[branchName]
}

func TestStackTreeRenderer_RenderStack_LinearStack(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	// Should have 3 branches: main, feature-1, feature-2
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Check that all branch names appear
	output := strings.Join(lines, "\n")
	for _, branch := range []string{"main", "feature-1", "feature-2"} {
		if !strings.Contains(output, branch) {
			t.Errorf("expected output to contain %q, got: %s", branch, output)
		}
	}
}

func TestStackTreeRenderer_RenderStack_WithAnnotations(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	prNum := 123
	renderer.SetAnnotation("feature-1", BranchAnnotation{
		PRNumber: &prNum,
		PRAction: "update",
	})

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	output := strings.Join(lines, "\n")
	// Should contain PR number
	if !strings.Contains(output, "#123") {
		t.Errorf("expected output to contain PR number #123, got: %s", output)
	}
}

func TestStackTreeRenderer_RenderStack_BranchingStack(t *testing.T) {
	mock := &mockTreeData{
		currentBranch: "feature-1a",
		trunk:         "main",
		children: map[string][]string{
			"main":       {"feature-1a", "feature-1b"},
			"feature-1a": {},
			"feature-1b": {},
		},
		parents: map[string]string{
			"feature-1a": "main",
			"feature-1b": "main",
		},
		fixed: map[string]bool{
			"main":       true,
			"feature-1a": true,
			"feature-1b": true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	// Should have 3 branches
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	output := strings.Join(lines, "\n")
	// Should contain branching characters for multiple children
	if !strings.Contains(output, "─") {
		t.Errorf("expected output to contain branching characters, got: %s", output)
	}
}

func TestStackTreeRenderer_RenderStack_FullFormat(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: false,
	})

	// Full format has more lines (branch line + trailing │ line for each)
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines in full format, got %d: %v", len(lines), lines)
	}

	output := strings.Join(lines, "\n")
	// Should contain the branch circle symbol
	if !strings.Contains(output, "◯") && !strings.Contains(output, "◉") {
		t.Errorf("expected output to contain circle symbols, got: %s", output)
	}
}

func TestStackTreeRenderer_RenderStack_Reversed(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	normalLines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	reversedLines := renderer.RenderStack("main", TreeRenderOptions{
		Short:   true,
		Reverse: true,
	})

	// Both should have same number of lines
	if len(normalLines) != len(reversedLines) {
		t.Errorf("expected same number of lines, got normal=%d reversed=%d", len(normalLines), len(reversedLines))
	}

	// First branch in normal should be last in reversed (approximately)
	normalOutput := strings.Join(normalLines, "\n")
	reversedOutput := strings.Join(reversedLines, "\n")

	// Both should contain all branches
	for _, branch := range []string{"main", "feature-1", "feature-2"} {
		if !strings.Contains(normalOutput, branch) {
			t.Errorf("normal output missing %q", branch)
		}
		if !strings.Contains(reversedOutput, branch) {
			t.Errorf("reversed output missing %q", branch)
		}
	}
}

func TestStackTreeRenderer_RenderBranchList(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	prNum := 42
	renderer.SetAnnotation("feature-1", BranchAnnotation{
		PRNumber: &prNum,
	})

	lines := renderer.RenderBranchList([]string{"feature-1", "feature-2"})

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "feature-1") || !strings.Contains(output, "feature-2") {
		t.Errorf("expected both branches in output, got: %s", output)
	}
}

func TestStackTreeRenderer_NeedsRestack(t *testing.T) {
	mock := &mockTreeData{
		currentBranch: "feature-1",
		trunk:         "main",
		children: map[string][]string{
			"main":      {"feature-1"},
			"feature-1": {},
		},
		parents: map[string]string{
			"feature-1": "main",
		},
		fixed: map[string]bool{
			"main":      true,
			"feature-1": false, // Not fixed - needs restack
		},
	}

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "needs restack") {
		t.Errorf("expected 'needs restack' indicator, got: %s", output)
	}
}

func TestBranchAnnotation_CheckStatus(t *testing.T) {
	mock := newMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.currentBranch,
		mock.trunk,
		mock.getChildren,
		mock.getParent,
		mock.isTrunk,
		mock.isBranchFixed,
	)

	renderer.SetAnnotation("feature-1", BranchAnnotation{
		CheckStatus: "PASSING",
	})
	renderer.SetAnnotation("feature-2", BranchAnnotation{
		CheckStatus: "FAILING",
	})

	lines := renderer.RenderStack("main", TreeRenderOptions{
		Short: true,
	})

	output := strings.Join(lines, "\n")
	// Should contain check icons
	if !strings.Contains(output, "✓") {
		t.Errorf("expected passing check icon ✓, got: %s", output)
	}
	if !strings.Contains(output, "✗") {
		t.Errorf("expected failing check icon ✗, got: %s", output)
	}
}
