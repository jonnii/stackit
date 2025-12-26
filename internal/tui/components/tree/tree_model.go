package tree

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model wraps StackTreeRenderer to make it a tea.Model for the storyboard
type Model struct {
	Renderer *StackTreeRenderer
	Options  RenderOptions
	Width    int
	Height   int
}

// NewModel creates a new Model with the given renderer.
func NewModel(renderer *StackTreeRenderer) *Model {
	return &Model{
		Renderer: renderer,
		Options: RenderOptions{
			Short: false,
		},
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update updates the model based on the message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.Options.Short = !m.Options.Short
			return m, nil
		case "r":
			m.Options.Reverse = !m.Options.Reverse
			return m, nil
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

// View returns the string representation of the model.
func (m Model) View() string {
	lines := m.Renderer.RenderStack(m.Renderer.trunk, m.Options)
	content := strings.Join(lines, "\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	help := helpStyle.Render("s: toggle short | r: toggle reverse | q: back")

	return content + "\n" + help
}
