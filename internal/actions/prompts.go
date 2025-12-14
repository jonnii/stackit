package actions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// textInputModel is a simple text input prompt model
type textInputModel struct {
	textInput textinput.Model
	prompt    string
	done      bool
	err       error
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("cancelled")
			m.done = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	if m.done {
		return ""
	}
	style := lipgloss.NewStyle().Margin(1, 0)
	return style.Render(fmt.Sprintf("%s\n%s\n\n(Press Enter to submit, Ctrl+C to cancel)", m.prompt, m.textInput.View()))
}

// confirmModel is a simple yes/no confirmation prompt model
type confirmModel struct {
	prompt string
	choice bool
	done   bool
	err    error
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("cancelled")
			m.done = true
			return m, tea.Quit
		case tea.KeyRunes:
			switch strings.ToLower(string(msg.Runes)) {
			case "y", "yes":
				m.choice = true
				m.done = true
				return m, tea.Quit
			case "n", "no":
				m.choice = false
				m.done = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	style := lipgloss.NewStyle().Margin(1, 0)
	yesNo := "[y/N]"
	if m.choice {
		yesNo = "[Y/n]"
	}
	return style.Render(fmt.Sprintf("%s %s\n\n(Press y/yes or n/no, Enter to confirm, Ctrl+C to cancel)", m.prompt, yesNo))
}

// promptTextInput prompts the user for text input
func promptTextInput(prompt, defaultValue string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.SetValue(defaultValue)
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	m := textInputModel{
		textInput: ti,
		prompt:    prompt,
	}

	p := tea.NewProgram(m, tea.WithOutput(nil))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := model.(textInputModel); ok {
		if m.err != nil {
			return "", m.err
		}
		return m.textInput.Value(), nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// promptConfirm prompts the user for yes/no confirmation
func promptConfirm(prompt string, defaultValue bool) (bool, error) {
	m := confirmModel{
		prompt: prompt,
		choice: defaultValue,
	}

	p := tea.NewProgram(m, tea.WithOutput(nil))
	_, err := p.Run()
	if err != nil {
		return false, err
	}

	if m.err != nil {
		return false, m.err
	}

	return m.choice, nil
}
