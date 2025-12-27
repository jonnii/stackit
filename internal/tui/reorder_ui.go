package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reorderKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	MoveUp   key.Binding
	MoveDown key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

func (k reorderKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.MoveUp, k.MoveDown, k.Confirm, k.Cancel}
}

func (k reorderKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.MoveUp, k.MoveDown},
		{k.Confirm, k.Cancel},
	}
}

var defaultReorderKeys = reorderKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	MoveUp: key.NewBinding(
		key.WithKeys("shift+up", "K"),
		key.WithHelp("K", "move up"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("shift+down", "J"),
		key.WithHelp("J", "move down"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+c", "q", "esc"),
		key.WithHelp("q/esc", "cancel"),
	),
}

// reorderModel is the bubbletea model for reordering branches
type reorderModel struct {
	branches  []string
	cursor    int
	confirmed bool
	canceled  bool
	styles    reorderStyles
	keys      reorderKeyMap
	help      help.Model
}

type reorderStyles struct {
	title    lipgloss.Style
	cursor   lipgloss.Style
	selected lipgloss.Style
	dim      lipgloss.Style
	branch   lipgloss.Style
}

func newReorderStyles() reorderStyles {
	return reorderStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).MarginBottom(1),
		cursor:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		branch:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	}
}

// newReorderModel creates a new reorder TUI model
func newReorderModel(branches []string) reorderModel {
	return reorderModel{
		branches: branches,
		cursor:   0,
		styles:   newReorderStyles(),
		keys:     defaultReorderKeys,
		help:     help.New(),
	}
}

func (m reorderModel) Init() tea.Cmd {
	return nil
}

func (m reorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Cancel):
			m.canceled = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}

		case key.Matches(msg, m.keys.MoveUp):
			if m.cursor > 0 {
				// Swap with previous
				m.branches[m.cursor], m.branches[m.cursor-1] = m.branches[m.cursor-1], m.branches[m.cursor]
				m.cursor--
			}

		case key.Matches(msg, m.keys.MoveDown):
			if m.cursor < len(m.branches)-1 {
				// Swap with next
				m.branches[m.cursor], m.branches[m.cursor+1] = m.branches[m.cursor+1], m.branches[m.cursor]
				m.cursor++
			}

		case key.Matches(msg, m.keys.Confirm):
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

	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	b.WriteString("\n")

	return b.String()
}

// NewReorderModel creates a tea.Model for a reorder prompt (used by stories/demo)
func NewReorderModel(branches []string) tea.Model {
	return newReorderModel(branches)
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
