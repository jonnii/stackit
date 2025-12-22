package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// reorderModel is the bubbletea model for reordering branches
type reorderModel struct {
	branches  []string
	cursor    int
	confirmed bool
	canceled  bool
	styles    reorderStyles
}

type reorderStyles struct {
	title       lipgloss.Style
	cursor      lipgloss.Style
	selected    lipgloss.Style
	dim         lipgloss.Style
	instruction lipgloss.Style
	branch      lipgloss.Style
}

func newReorderStyles() reorderStyles {
	return reorderStyles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).MarginBottom(1),
		cursor:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		selected:    lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		instruction: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginTop(1),
		branch:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	}
}

// newReorderModel creates a new reorder TUI model
func newReorderModel(branches []string) reorderModel {
	return reorderModel{
		branches: branches,
		cursor:   0,
		styles:   newReorderStyles(),
	}
}

func (m reorderModel) Init() tea.Cmd {
	return nil
}

func (m reorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case KeyCtrlC, KeyQuit, KeyEsc:
			m.canceled = true
			return m, tea.Quit

		case KeyUp, "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case KeyDown, "j":
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}

		case "shift+up", "K":
			if m.cursor > 0 {
				// Swap with previous
				m.branches[m.cursor], m.branches[m.cursor-1] = m.branches[m.cursor-1], m.branches[m.cursor]
				m.cursor--
			}

		case "shift+down", "J":
			if m.cursor < len(m.branches)-1 {
				// Swap with next
				m.branches[m.cursor], m.branches[m.cursor+1] = m.branches[m.cursor+1], m.branches[m.cursor]
				m.cursor++
			}

		case KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m reorderModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("Reorder Branches"))
	b.WriteString("\n")

	for i, branch := range m.branches {
		cursor := "  "
		style := m.styles.branch
		if i == m.cursor {
			cursor = m.styles.cursor.Render("▸ ")
			style = m.styles.selected
		}

		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(branch)))
	}

	b.WriteString(m.styles.instruction.Render("↑/↓: navigate • Shift+↑/↓ (K/J): move • Enter: confirm • q/Esc: cancel"))
	b.WriteString("\n")

	return b.String()
}

// RunReorderTUI runs the reorder TUI and returns the new order
func RunReorderTUI(branches []string) ([]string, error) {
	m := newReorderModel(branches)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	res := finalModel.(reorderModel)
	if res.canceled {
		return nil, fmt.Errorf("reorder canceled")
	}

	return res.branches, nil
}
