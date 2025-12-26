// Package main is the entry point for the st-tui application.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui"
)

type state int

const (
	stateList state = iota
	stateStory
)

type model struct {
	state       state
	cursor      int
	stories     []tui.Story
	activeStory *tui.Story
	storyModel  tea.Model
	width       int
	height      int
}

func main() {
	m := model{
		state:   stateList,
		stories: tui.Stories,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running storyboard: %v\n", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global handlers
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.state == stateStory {
		// Handle global back-to-list keys
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "q" || k.String() == "esc" {
				m.state = stateList
				m.activeStory = nil
				m.storyModel = nil
				return m, nil
			}
		}

		// Pass ALL messages to the active story
		var cmd tea.Cmd
		m.storyModel, cmd = m.storyModel.Update(msg)
		return m, cmd
	}

	// List mode handling
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.stories)-1 {
				m.cursor++
			}
		case "enter":
			m.state = stateStory
			m.activeStory = &m.stories[m.cursor]
			m.storyModel = m.activeStory.CreateModel()
			return m, m.storyModel.Init()
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.state == stateStory {
		return m.storyModel.View()
	}

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Stackit TUI Storyboard"))
	b.WriteString("\n\n")

	for i, story := range m.stories {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "> "
			style = style.Foreground(lipgloss.Color("5")).Bold(true)
		}

		b.WriteString(cursor)
		b.WriteString(style.Render(fmt.Sprintf("[%s] %s", story.Category, story.Name)))
		b.WriteString("\n")
		if i == m.cursor {
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginLeft(4)
			b.WriteString(descStyle.Render(story.Description))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑/↓: navigate | enter: view | q: quit"))

	return lipgloss.NewStyle().Margin(1, 2).Render(b.String())
}
