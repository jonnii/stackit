package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/tui/components/tree"
)

// Story represents a specific state of a TUI component
type Story struct {
	Name        string
	Category    string
	Description string
	CreateModel func() tea.Model
}

// Stories is a registry of all component stories
var Stories = []Story{}

// RegisterStory registers a new component story
func RegisterStory(story Story) {
	Stories = append(Stories, story)
}

func init() {
	registerTreeStories()
}

func registerTreeStories() {
	RegisterStory(Story{
		Name:        "Linear Stack",
		Category:    "Tree",
		Description: "A simple 3-branch linear stack with some PR annotations",
		CreateModel: func() tea.Model {
			mock := &tree.MockTreeData{
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
			renderer := tree.NewStackTreeRenderer(mock.CurrentBranch, mock.Trunk, mock.GetChildren, mock.GetParent, mock.IsTrunk, mock.IsBranchFixed)

			pr1 := 101
			renderer.SetAnnotation("feature-1", tree.BranchAnnotation{
				PRNumber:     &pr1,
				Scope:        "API",
				CommitCount:  2,
				LinesAdded:   50,
				LinesDeleted: 10,
				CheckStatus:  "PASSING",
			})

			pr2 := 102
			renderer.SetAnnotation("feature-2", tree.BranchAnnotation{
				PRNumber:     &pr2,
				Scope:        "UI",
				CommitCount:  5,
				LinesAdded:   120,
				LinesDeleted: 5,
				CheckStatus:  "PENDING",
			})

			return tree.NewModel(renderer)
		},
	})

	RegisterStory(Story{
		Name:        "Complex Branching",
		Category:    "Tree",
		Description: "A tree with multiple branches at the same level and different scopes",
		CreateModel: func() tea.Model {
			mock := &tree.MockTreeData{
				CurrentBranch: "auth-fix",
				Trunk:         "main",
				Children: map[string][]string{
					"main":      {"base-api", "base-auth"},
					"base-api":  {"api-v2", "api-docs"},
					"base-auth": {"auth-fix"},
					"api-v2":    {},
					"api-docs":  {},
					"auth-fix":  {},
				},
				Parents: map[string]string{
					"base-api":  "main",
					"base-auth": "main",
					"api-v2":    "base-api",
					"api-docs":  "base-api",
					"auth-fix":  "base-auth",
				},
				Fixed: map[string]bool{
					"main":      true,
					"base-api":  true,
					"base-auth": true,
					"api-v2":    true,
					"api-docs":  false, // needs restack
					"auth-fix":  true,
				},
			}
			renderer := tree.NewStackTreeRenderer(mock.CurrentBranch, mock.Trunk, mock.GetChildren, mock.GetParent, mock.IsTrunk, mock.IsBranchFixed)

			renderer.SetAnnotation("base-api", tree.BranchAnnotation{Scope: "API", ExplicitScope: "API", CommitCount: 1})
			renderer.SetAnnotation("api-v2", tree.BranchAnnotation{Scope: "API", CommitCount: 10, LinesAdded: 400})
			renderer.SetAnnotation("api-docs", tree.BranchAnnotation{Scope: "API", CommitCount: 1, LinesAdded: 20})

			renderer.SetAnnotation("base-auth", tree.BranchAnnotation{Scope: "AUTH", ExplicitScope: "AUTH", CommitCount: 1})
			renderer.SetAnnotation("auth-fix", tree.BranchAnnotation{Scope: "AUTH", CommitCount: 3, LinesAdded: 30, LinesDeleted: 30, CheckStatus: "FAILING"})

			return tree.NewModel(renderer)
		},
	})
}
