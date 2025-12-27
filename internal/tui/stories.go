package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/components/submit"
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
	registerSubmitStories()
	registerMergeStories()
	registerPromptStories()
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

	RegisterStory(Story{
		Name:        "Stack Submission",
		Category:    "Tree",
		Description: "The configuration view before submitting a stack, showing planned actions",
		CreateModel: func() tea.Model {
			mock := &tree.MockTreeData{
				CurrentBranch: "feature-3",
				Trunk:         "main",
				Children: map[string][]string{
					"main":      {"feature-1"},
					"feature-1": {"feature-2"},
					"feature-2": {"feature-3"},
					"feature-3": {},
				},
				Parents: map[string]string{
					"feature-1": "main",
					"feature-2": "feature-1",
					"feature-3": "feature-2",
				},
				Fixed: map[string]bool{
					"main":      true,
					"feature-1": true,
					"feature-2": true,
					"feature-3": true,
				},
			}
			renderer := tree.NewStackTreeRenderer(mock.CurrentBranch, mock.Trunk, mock.GetChildren, mock.GetParent, mock.IsTrunk, mock.IsBranchFixed)

			pr1 := 101
			renderer.SetAnnotation("feature-1", tree.BranchAnnotation{
				PRNumber:      &pr1,
				PRAction:      "skip",
				CustomLabel:   "(up to date)",
				Scope:         "CORE",
				ExplicitScope: "CORE",
				CommitCount:   0,
			})

			pr2 := 102
			renderer.SetAnnotation("feature-2", tree.BranchAnnotation{
				PRNumber:      &pr2,
				PRAction:      "update",
				Scope:         "API",
				ExplicitScope: "API",
				CommitCount:   3,
				LinesAdded:    45,
				LinesDeleted:  12,
			})

			renderer.SetAnnotation("feature-3", tree.BranchAnnotation{
				PRAction:      "create",
				Scope:         "UI",
				ExplicitScope: "UI",
				CommitCount:   5,
				LinesAdded:    120,
				LinesDeleted:  5,
			})

			model := tree.NewModel(renderer)
			model.Options.HideStats = true
			return model
		},
	})
}

func registerSubmitStories() {
	mockData := &tree.MockTreeData{
		CurrentBranch: "feature-3",
		Trunk:         "main",
		Children: map[string][]string{
			"main":      {"feature-1"},
			"feature-1": {"feature-2"},
			"feature-2": {"feature-3"},
			"feature-3": {},
		},
		Parents: map[string]string{
			"feature-1": "main",
			"feature-2": "feature-1",
			"feature-3": "feature-2",
		},
		Fixed: map[string]bool{
			"main":      true,
			"feature-1": true,
			"feature-2": true,
			"feature-3": true,
		},
	}

	createRenderer := func() *tree.StackTreeRenderer {
		return tree.NewStackTreeRenderer(mockData.CurrentBranch, mockData.Trunk, mockData.GetChildren, mockData.GetParent, mockData.IsTrunk, mockData.IsBranchFixed)
	}

	RegisterStory(Story{
		Name:        "Full Submission",
		Category:    "Submit",
		Description: "A simulated full submission process with state transitions",
		CreateModel: func() tea.Model {
			m := submit.NewModel(nil)
			m.Renderer = createRenderer()
			m.Renderer.SetAnnotation("feature-1", tree.BranchAnnotation{Scope: "CORE", ExplicitScope: "CORE"})
			m.Renderer.SetAnnotation("feature-2", tree.BranchAnnotation{Scope: "API", ExplicitScope: "API"})
			m.Renderer.SetAnnotation("feature-3", tree.BranchAnnotation{Scope: "UI", ExplicitScope: "UI"})
			m.RootBranch = mockData.Trunk
			return &submitSimulationModel{
				submitModel: m,
				startTime:   time.Now(),
			}
		},
	})

	RegisterStory(Story{
		Name:        "Submission Error",
		Category:    "Submit",
		Description: "A submission with an error on one of the branches",
		CreateModel: func() tea.Model {
			m := submit.NewModel([]submit.Item{
				{BranchName: "feature-1", Action: "update", Status: submit.StatusDone, URL: "https://github.com/owner/repo/pull/101"},
				{BranchName: "feature-2", Action: "update", Status: submit.StatusError, Error: fmt.Errorf("failed to push branch: remote rejected")},
				{BranchName: "feature-3", Action: "create", Status: submit.StatusPending},
			})
			m.Renderer = createRenderer()
			m.Renderer.SetAnnotation("feature-1", tree.BranchAnnotation{Scope: "CORE", ExplicitScope: "CORE"})
			m.Renderer.SetAnnotation("feature-2", tree.BranchAnnotation{Scope: "API", ExplicitScope: "API"})
			m.Renderer.SetAnnotation("feature-3", tree.BranchAnnotation{Scope: "UI", ExplicitScope: "UI"})
			m.RootBranch = mockData.Trunk
			m.GlobalMessage = "Submitting 3 branches..."
			m.Done = true
			return m
		},
	})
}

type submitSimulationModel struct {
	submitModel *submit.Model
	step        int
	startTime   time.Time
}

func (m *submitSimulationModel) Init() tea.Cmd {
	return tea.Batch(
		m.submitModel.Init(),
		m.nextTick(),
	)
}

func (m *submitSimulationModel) nextTick() tea.Cmd {
	delay := 1 * time.Second
	if m.step == 0 {
		delay = 100 * time.Millisecond
	}
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return simulationTickMsg(m.step)
	})
}

type simulationTickMsg int

func (m *submitSimulationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case simulationTickMsg:
		if int(msg) == -1 {
			// Reset simulation
			m.step = 0
			m.submitModel = submit.NewModel(nil)
			m.submitModel.Renderer = tree.NewStackTreeRenderer(
				"feature-3", "main",
				func(b string) []string {
					children := map[string][]string{
						"main":      {"feature-1"},
						"feature-1": {"feature-2"},
						"feature-2": {"feature-3"},
						"feature-3": {},
					}
					return children[b]
				},
				func(b string) string {
					parents := map[string]string{
						"feature-1": "main",
						"feature-2": "feature-1",
						"feature-3": "feature-2",
					}
					return parents[b]
				},
				func(b string) bool { return b == "main" },
				func(_ string) bool { return true },
			)
			m.submitModel.Renderer.SetAnnotation("feature-1", tree.BranchAnnotation{Scope: "CORE", ExplicitScope: "CORE"})
			m.submitModel.Renderer.SetAnnotation("feature-2", tree.BranchAnnotation{Scope: "API", ExplicitScope: "API"})
			m.submitModel.Renderer.SetAnnotation("feature-3", tree.BranchAnnotation{Scope: "UI", ExplicitScope: "UI"})
			m.submitModel.RootBranch = "main"
			return m, m.nextTick()
		}

		m.step++
		var cmds []tea.Cmd

		switch m.step {
		case 1:
			_, c := m.submitModel.Update(submit.GlobalMessageMsg("Preparing branches..."))
			cmds = append(cmds, c, m.nextTick())
		case 2:
			m.submitModel.Update(submit.PlanUpdateMsg{BranchName: "feature-1", Action: "update", Skip: true, SkipReason: "already up to date"})
			m.submitModel.Update(submit.PlanUpdateMsg{BranchName: "feature-2", Action: "update"})
			_, c := m.submitModel.Update(submit.PlanUpdateMsg{BranchName: "feature-3", Action: "create"})
			cmds = append(cmds, c, m.nextTick())
		case 3:
			m.submitModel.Update(submit.GlobalMessageMsg("Submitting..."))
			_, c := m.submitModel.Update(submit.ProgressUpdateMsg{BranchName: "feature-2", Status: submit.StatusSubmitting})
			cmds = append(cmds, c, m.nextTick())
		case 4:
			m.submitModel.Update(submit.ProgressUpdateMsg{BranchName: "feature-2", Status: submit.StatusDone, URL: "https://github.com/owner/repo/pull/102"})
			_, c := m.submitModel.Update(submit.ProgressUpdateMsg{BranchName: "feature-3", Status: submit.StatusSubmitting})
			cmds = append(cmds, c, m.nextTick())
		case 5:
			_, c := m.submitModel.Update(submit.ProgressUpdateMsg{BranchName: "feature-3", Status: submit.StatusDone, URL: "https://github.com/owner/repo/pull/103"})
			cmds = append(cmds, c, m.nextTick())
			m.submitModel.Update(submit.GlobalMessageMsg("✓ All branches submitted"))
			// We don't send ProgressCompleteMsg because it would trigger tea.Quit
			m.submitModel.Done = true
			cmds = append(cmds, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
				return simulationTickMsg(-1) // Signal reset
			}))
		}

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	newModel, cmd := m.submitModel.Update(msg)
	m.submitModel = newModel.(*submit.Model)
	return m, cmd
}

func (m *submitSimulationModel) View() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	return m.submitModel.View() + "\n" +
		helpStyle.Render(fmt.Sprintf("Simulation step: %d/6", m.step)) + "\n" +
		helpStyle.Render("q: back")
}
