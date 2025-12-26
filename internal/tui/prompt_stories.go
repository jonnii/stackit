package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func registerPromptStories() {
	RegisterStory(Story{
		Name:        "Confirmation Prompt",
		Category:    "Prompt",
		Description: "A simple yes/no confirmation prompt",
		CreateModel: func() tea.Model {
			return NewConfirmModel("Are you sure you want to delete this branch?", false)
		},
	})

	RegisterStory(Story{
		Name:        "Text Input Prompt",
		Category:    "Prompt",
		Description: "A text input prompt for entering a value",
		CreateModel: func() tea.Model {
			return NewTextInputModel("Enter the new branch name:", "feature-branch")
		},
	})

	RegisterStory(Story{
		Name:        "Option Selector",
		Category:    "Prompt",
		Description: "A searchable list of options",
		CreateModel: func() tea.Model {
			return NewSelectModel("Choose an action:", []SelectOption{
				{Label: "🚀 Deploy to production", Value: "deploy"},
				{Label: "🧪 Run tests", Value: "test"},
				{Label: "🧹 Clean workspace", Value: "clean"},
				{Label: "📝 Edit configuration", Value: "edit"},
			}, 0)
		},
	})

	RegisterStory(Story{
		Name:        "Branch Selector",
		Category:    "Prompt",
		Description: "A searchable list of branches with tree visualization",
		CreateModel: func() tea.Model {
			return NewBranchSelectModel("Select a branch to checkout:", []BranchChoice{
				{Display: "◉ main", Value: "main"},
				{Display: "  ◯ feature-1", Value: "feature-1"},
				{Display: "    ◯ feature-2", Value: "feature-2"},
				{Display: "  ◯ feature-3", Value: "feature-3"},
			}, 1)
		},
	})

	RegisterStory(Story{
		Name:        "Branch Reordering",
		Category:    "Reorder",
		Description: "Interactive interface for reordering branches in a stack",
		CreateModel: func() tea.Model {
			return NewReorderModel([]string{
				"feature-1 (base)",
				"feature-2 (api)",
				"feature-3 (ui)",
				"feature-4 (docs)",
			})
		},
	})
}
