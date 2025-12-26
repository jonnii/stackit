package tree

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"stackit.dev/stackit/internal/tui/style"
)

func init() {
	// Force color output for all tests in this file to ensure ANSI escape codes are generated
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestStackTreeRenderer_RenderStack_LinearStack(t *testing.T) {
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	lines := renderer.RenderStack("main", RenderOptions{
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
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	prNum := 123
	renderer.SetAnnotation("feature-1", BranchAnnotation{
		PRNumber: &prNum,
		PRAction: "update",
	})

	lines := renderer.RenderStack("main", RenderOptions{
		Short: true,
	})

	output := strings.Join(lines, "\n")
	// Should contain PR number
	if !strings.Contains(output, "#123") {
		t.Errorf("expected output to contain PR number #123, got: %s", output)
	}
}

func TestStackTreeRenderer_RenderStack_BranchingStack(t *testing.T) {
	mock := &MockTreeData{
		CurrentBranch: "feature-1a",
		Trunk:         "main",
		Children: map[string][]string{
			"main":       {"feature-1a", "feature-1b"},
			"feature-1a": {},
			"feature-1b": {},
		},
		Parents: map[string]string{
			"feature-1a": "main",
			"feature-1b": "main",
		},
		Fixed: map[string]bool{
			"main":       true,
			"feature-1a": true,
			"feature-1b": true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	lines := renderer.RenderStack("main", RenderOptions{
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
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	lines := renderer.RenderStack("main", RenderOptions{
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
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	normalLines := renderer.RenderStack("main", RenderOptions{
		Short: true,
	})

	reversedLines := renderer.RenderStack("main", RenderOptions{
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
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
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
	mock := &MockTreeData{
		CurrentBranch: "feature-1",
		Trunk:         "main",
		Children: map[string][]string{
			"main":      {"feature-1"},
			"feature-1": {},
		},
		Parents: map[string]string{
			"feature-1": "main",
		},
		Fixed: map[string]bool{
			"main":      true,
			"feature-1": false, // Not fixed - needs restack
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	lines := renderer.RenderStack("main", RenderOptions{
		Short: true,
	})

	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "needs restack") {
		t.Errorf("expected 'needs restack' indicator, got: %s", output)
	}
}

func TestBranchAnnotation_CheckStatus(t *testing.T) {
	mock := NewMockTreeData()

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	renderer.SetAnnotation("feature-1", BranchAnnotation{
		CheckStatus: "PASSING",
	})
	renderer.SetAnnotation("feature-2", BranchAnnotation{
		CheckStatus: "FAILING",
	})

	lines := renderer.RenderStack("main", RenderOptions{
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

func TestStackTreeRenderer_ScopeColoring(t *testing.T) {
	mock := &MockTreeData{
		CurrentBranch: "feature-login",
		Trunk:         "main",
		Children: map[string][]string{
			"main":              {"feature-auth-base", "feature-api-v1"},
			"feature-auth-base": {"feature-login"},
			"feature-api-v1":    {},
			"feature-login":     {},
		},
		Parents: map[string]string{
			"feature-auth-base": "main",
			"feature-login":     "feature-auth-base",
			"feature-api-v1":    "main",
		},
		Fixed: map[string]bool{
			"main":              true,
			"feature-auth-base": true,
			"feature-login":     true,
			"feature-api-v1":    true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	// Set scopes
	renderer.SetAnnotation("feature-auth-base", BranchAnnotation{Scope: "AUTH", ExplicitScope: "AUTH"})
	renderer.SetAnnotation("feature-login", BranchAnnotation{Scope: "AUTH"})
	renderer.SetAnnotation("feature-api-v1", BranchAnnotation{Scope: "API", ExplicitScope: "API"})

	lines := renderer.RenderStack("main", RenderOptions{
		Short: false,
	})

	output := strings.Join(lines, "\n")

	// Get expected colors
	authHex, _ := style.GetScopeColor("AUTH")

	// Verify that the scope labels are present and colored
	if !strings.Contains(output, "[AUTH]") {
		t.Errorf("expected output to contain [AUTH]")
	}
	if !strings.Contains(output, "[API]") {
		t.Errorf("expected output to contain [API]")
	}

	// Verify that tree lines are colored
	authStyle := lipgloss.NewStyle().Foreground(authHex).Render("│")

	if !strings.Contains(output, authStyle) {
		t.Errorf("expected output to contain colored vertical line for AUTH scope")
	}
}

func TestStackTreeRenderer_InheritedScopeColoring(t *testing.T) {
	mock := &MockTreeData{
		CurrentBranch: "feature-login",
		Trunk:         "main",
		Children: map[string][]string{
			"main":              {"feature-auth-base"},
			"feature-auth-base": {"feature-login"},
			"feature-login":     {},
		},
		Parents: map[string]string{
			"feature-auth-base": "main",
			"feature-login":     "feature-auth-base",
		},
		Fixed: map[string]bool{
			"main":              true,
			"feature-auth-base": true,
			"feature-login":     true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	// Set scope only on the base branch
	renderer.SetAnnotation("feature-auth-base", BranchAnnotation{Scope: "AUTH", ExplicitScope: "AUTH"})
	// Inherited scope on child, but NO ExplicitScope
	renderer.SetAnnotation("feature-login", BranchAnnotation{Scope: "AUTH"})

	lines := renderer.RenderStack("main", RenderOptions{
		Short: false,
	})

	output := strings.Join(lines, "\n")

	// Get expected colors
	authHex, _ := style.GetScopeColor("AUTH")

	// Verify that the scope label is present only for the base branch
	if !strings.Contains(output, "feature-auth-base") || !strings.Contains(output, "[AUTH]") {
		t.Errorf("expected base branch to have [AUTH] label")
	}

	// feature-login should NOT have the [AUTH] label but its symbol and tree lines should be colored
	loginLine := ""
	trailingLine := ""
	for i, line := range lines {
		if strings.Contains(line, "feature-login") {
			loginLine = line
			if i+1 < len(lines) {
				trailingLine = lines[i+1]
			}
			break
		}
	}

	if loginLine == "" {
		t.Fatalf("could not find line for feature-login")
	}

	if strings.Contains(loginLine, "[AUTH]") {
		t.Errorf("expected inherited branch to NOT have [AUTH] label")
	}

	authStyleSymbol := lipgloss.NewStyle().Foreground(authHex).Render(CurrentBranchSymbol)
	if !strings.Contains(loginLine, authStyleSymbol) {
		t.Errorf("expected inherited branch symbol to be colored with AUTH scope. Line was: %q", loginLine)
	}

	authStyleVertical := lipgloss.NewStyle().Foreground(authHex).Render("│")
	if !strings.Contains(trailingLine, authStyleVertical) {
		t.Errorf("expected trailing line to be colored with AUTH scope. Line was: %q", trailingLine)
	}
}

func TestStackTreeRenderer_BranchingScopeColoring(t *testing.T) {
	mock := &MockTreeData{
		CurrentBranch: "feature-1a",
		Trunk:         "main",
		Children: map[string][]string{
			"main":       {"feature-1a", "feature-1b"},
			"feature-1a": {},
			"feature-1b": {},
		},
		Parents: map[string]string{
			"feature-1a": "main",
			"feature-1b": "main",
		},
		Fixed: map[string]bool{
			"main":       true,
			"feature-1a": true,
			"feature-1b": true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	// Set scope on main
	renderer.SetAnnotation("main", BranchAnnotation{Scope: "AUTH", ExplicitScope: "AUTH"})
	renderer.SetAnnotation("feature-1a", BranchAnnotation{Scope: "AUTH"})
	renderer.SetAnnotation("feature-1b", BranchAnnotation{Scope: "AUTH"})

	lines := renderer.RenderStack("main", RenderOptions{
		Short: false,
	})

	output := strings.Join(lines, "\n")

	// Get expected color for AUTH
	authHex, _ := style.GetScopeColor("AUTH")
	authStyleBranch := lipgloss.NewStyle().Foreground(authHex).Render("├──┘")

	if !strings.Contains(output, authStyleBranch) {
		t.Errorf("expected output to contain colored branching characters for AUTH scope. Output was:\n%s", output)
	}
}

func TestStackTreeRenderer_ScopeColoringBoundaries(t *testing.T) {
	// Setup a branching structure where one branch has a scope
	// main
	//   └─ base (A)
	//       ├─ scoped-branch (B) [SCOPE-X]
	//       └─ unscoped-branch (C)
	mock := &MockTreeData{
		CurrentBranch: "scoped-branch",
		Trunk:         "main",
		Children: map[string][]string{
			"main": {"base"},
			"base": {
				"scoped-branch",
				"unscoped-branch",
			},
			"scoped-branch":   {},
			"unscoped-branch": {},
		},
		Parents: map[string]string{
			"base":            "main",
			"scoped-branch":   "base",
			"unscoped-branch": "base",
		},
		Fixed: map[string]bool{
			"main":            true,
			"base":            true,
			"scoped-branch":   true,
			"unscoped-branch": true,
		},
	}

	renderer := NewStackTreeRenderer(
		mock.CurrentBranch,
		mock.Trunk,
		mock.GetChildren,
		mock.GetParent,
		mock.IsTrunk,
		mock.IsBranchFixed,
	)

	// Only scoped-branch has the scope
	renderer.SetAnnotation("scoped-branch", BranchAnnotation{
		Scope:         "SCOPE-X",
		ExplicitScope: "SCOPE-X",
	})
	renderer.SetAnnotation("base", BranchAnnotation{Scope: ""})
	renderer.SetAnnotation("unscoped-branch", BranchAnnotation{Scope: ""})

	lines := renderer.RenderStack("main", RenderOptions{
		Short: false,
	})

	// Get colors
	scopeXColor, _ := style.GetScopeColor("SCOPE-X")
	scopeXStyle := lipgloss.NewStyle().Foreground(scopeXColor)
	scopeXLine := scopeXStyle.Render("│")

	// 1. unscoped-branch should NOT have the scope color in its vertical line
	for _, line := range lines {
		if strings.Contains(line, "unscoped-branch") {
			if strings.Contains(line, scopeXLine) {
				t.Errorf("unscoped-branch line should not contain SCOPE-X color. Line: %q", line)
			}
		}
	}

	// 2. The vertical line connecting scoped-branch to its UNSCOPED parent should NOT be colored
	foundScoped := false
	for i, line := range lines {
		if strings.Contains(line, "scoped-branch") {
			foundScoped = true
			if i+1 < len(lines) {
				nextLine := lines[i+1] // Vertical line below scoped-branch
				if strings.Contains(nextLine, scopeXLine) {
					t.Errorf("Vertical line connecting scoped-branch to unscoped parent should not be colored. Line: %q", nextLine)
				}
			}
		}
	}
	if !foundScoped {
		t.Error("scoped-branch not found in output")
	}

	// 3. The symbol for scoped-branch SHOULD be colored
	scopeXSymbol := scopeXStyle.Render(CurrentBranchSymbol)
	foundSymbol := false
	for _, line := range lines {
		if strings.Contains(line, "scoped-branch") {
			if strings.Contains(line, scopeXSymbol) {
				foundSymbol = true
			}
		}
	}
	if !foundSymbol {
		t.Error("scoped-branch symbol should be colored")
	}
}
